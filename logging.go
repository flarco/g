package g

import (
	jsonGo "encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/rs/zerolog"
	"github.com/spf13/cast"
)

// LogHook is a hook to be perform at the specified level
type LogHook struct {
	Level zerolog.Level
	Send  func(level zerolog.Level, t string, a ...interface{})
	Func  func(*LogLine)
}

const (
	ColorBlack = iota + 30
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite

	ColorBold     = 1
	ColorDarkGray = 90
)

// LogHooks are log hooks
var LogHooks = []*LogHook{}

// LogLevel is the log level
var LogLevel = new(Level)

// ZLogOut is the non-error/normal logger
var ZLogOut = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05"}).With().Timestamp().Logger()

// ZLogErr is the error/debug logger
var ZLogErr = zerolog.New(zerolog.ConsoleWriter{
	Out:        os.Stderr,
	TimeFormat: "2006-01-02 15:04:05",
	FormatLevel: func(i interface{}) string {
		var levelColor string
		if ll, ok := i.(string); ok {
			switch ll {
			case zerolog.LevelTraceValue:
				levelColor = Colorize(ColorMagenta, "TRC")
			case zerolog.LevelDebugValue:
				levelColor = Colorize(ColorYellow, "DBG")
			case zerolog.LevelInfoValue:
				levelColor = Colorize(ColorGreen, "INF")
			case zerolog.LevelWarnValue:
				levelColor = Colorize(ColorRed, "WRN")
			case zerolog.LevelErrorValue:
				levelColor = Colorize(ColorRed, "ERR")
			case zerolog.LevelFatalValue:
				levelColor = Colorize(ColorRed, "FTL")
			case zerolog.LevelPanicValue:
				levelColor = Colorize(ColorRed, "PNC")
			default:
				levelColor = ll
			}
		}
		return levelColor + " "
	},
}).With().Timestamp().Logger()

// CallerLevel is the stack caller information level
var CallerLevel = 0

// DisableColor disables color
var DisableColor = false

// Level is the log level
type Level int8

const (
	// TraceLevel defines trace log level.
	TraceLevel Level = iota
	// LowDebugLevel defines low debug log level.
	LowDebugLevel
	// DebugLevel defines debug log level.
	DebugLevel
	// NormalLevel defines normal log level.
	NormalLevel
	// WarnLevel defines warning log level.
	WarnLevel
)

func init() {
	SetZeroLogLevel(zerolog.InfoLevel)
	SetLogLevel(NormalLevel)
	GetLogLevel()
	// CallerLevel = 2
	if os.Getenv("_DEBUG_CALLER_LEVEL") != "" {
		CallerLevel = cast.ToInt(os.Getenv("_DEBUG_CALLER_LEVEL"))
	}
}

// SetZeroLogLevel sets the zero log level
func SetZeroLogLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
}

// SetLogLevel sets the g log level
func SetLogLevel(level Level) {
	LogLevel = &level
}

type LogLine struct {
	Time  time.Time `json:"time,omitempty"`
	Level int8      `json:"level,omitempty"`
	Group string    `json:"group,omitempty"`
	Text  string    `json:"text,omitempty"`
	Args  []any     `json:"args,omitempty"`
}

func (ll *LogLine) Line() string {
	// construct log line like zerolog
	var timeText, levelPrefix string

	switch zerolog.Level(ll.Level) {
	case zerolog.TraceLevel:
		levelPrefix = "\x1b[35mTRC\x1b[0m "
	case zerolog.DebugLevel:
		levelPrefix = "\x1b[33mDBG\x1b[0m "
	case zerolog.InfoLevel:
		levelPrefix = "\x1b[32mINF\x1b[0m "
	case zerolog.WarnLevel:
		levelPrefix = "\x1b[31mWRN\x1b[0m "
	}

	if !ll.Time.IsZero() {
		timeText = F(
			"\x1b[90m%s\x1b[0m ",
			ll.Time.Format("2006-01-02 15:04:05"),
		)
	}

	return F(timeText+levelPrefix+ll.Text, ll.Args...)
}

type LogLines []LogLine

func (lls LogLines) Lines() (lines []string) {
	lines = make([]string, len(lls))
	for _, ll := range lls {
		lines = append(lines, ll.Line())
	}
	return lines
}

// NewLogHook return a new log hook
func NewLogHook(level Level, f func(*LogLine)) *LogHook {
	zLevel := zerolog.InfoLevel
	switch level {
	case TraceLevel:
		zLevel = zerolog.TraceLevel
	case DebugLevel, LowDebugLevel:
		zLevel = zerolog.DebugLevel
	case WarnLevel:
		zLevel = zerolog.WarnLevel
	}

	return &LogHook{
		Level: zLevel,
		Func:  f,
	}
}

// GetLogLevel gets the g log level
func GetLogLevel() Level {
	// legacy setting
	if val := os.Getenv("_DEBUG"); val != "" {
		os.Setenv("DEBUG", val)
	}

	if val := os.Getenv("DEBUG"); val != "" {
		switch val {
		case "TRACE":
			SetZeroLogLevel(zerolog.TraceLevel)
			SetLogLevel(TraceLevel)
		case "LOW":
			SetZeroLogLevel(zerolog.DebugLevel)
			SetLogLevel(LowDebugLevel)
		case "DEBUG":
			SetLogLevel(DebugLevel)
			SetZeroLogLevel(zerolog.DebugLevel)
		}
	}
	return *LogLevel
}

// SetZeroLogHook sets a zero log hook
func SetZeroLogHook(h zerolog.Hook) {
	ZLogOut = ZLogOut.Hook(h)
	ZLogErr = ZLogErr.Hook(h)
}

// SetLogHook sets a log hook
func SetLogHook(lh *LogHook) {
	LogHooks = append(LogHooks, lh)
}

// RemoveLogHook removes a log hook
func RemoveLogHook(lh *LogHook) {
	for i, hook := range LogHooks {
		if hook == lh {
			LogHooks = append(LogHooks[:i], LogHooks[i+1:]...)
			break
		}
	}
}

// IsDebug returns true is debug is low
func IsDebug() bool {
	return GetLogLevel() == DebugLevel || IsDebugLow()
}

// IsDebugLow returns true is debug is low
func IsDebugLow() bool {
	return GetLogLevel() == LowDebugLevel || GetLogLevel() == TraceLevel
}

// IsTrace returns true is debug is trace
func IsTrace() bool {
	return GetLogLevel() == TraceLevel
}

func disableColor() bool {
	return DisableColor
}

// PP prints the Pretty Printed JSON struct
func PP(v interface{}) {
	if IsDebugLow() {
		args := addCaller([]interface{}{})
		doLog(ZLogErr.Debug(), Pretty(v), args)
	}
}

// Pretty returns the Pretty Printed JSON struct string
func Pretty(v interface{}) string {
	vv, _ := jsonGo.MarshalIndent(v, "", "  ")
	return string(vv)
}

// P prints the value of object
func P(v interface{}) {
	if IsDebugLow() {
		args := addCaller([]interface{}{})
		doLog(ZLogErr.Debug(), F("%#v", v), args)
	}
}

// extractLogMapArgs remove the map[string]interface arguments
// and places them as context field/values in the localLog
func extractLogMapArgs(args []interface{}, localLog *zerolog.Event) []interface{} {
	newArgs := []interface{}{}
	for _, val := range args {
		switch val.(type) {
		case map[string]interface{}:
			mapInterf := val.(map[string]interface{})
			for k, v := range mapInterf {
				localLog = localLog.Interface(k, v)
			}
		default:
			newArgs = append(newArgs, val)
		}
	}
	return newArgs
}

// Log : print text
func Log(text string, args ...interface{}) {
	localLogDbg := ZLogErr.Debug()
	localLogInf := ZLogOut.Info()
	extractLogMapArgs(args, localLogDbg)
	args = extractLogMapArgs(args, localLogInf)
	text = F(text, args...)

	if strings.HasPrefix(text, "g+") {
		LogC(text[2:], "green", os.Stderr)
	} else if strings.HasPrefix(text, "+") {
		LogC(text[1:], "green", os.Stderr)
	} else if strings.HasPrefix(text, "b+") {
		LogC(text[2:], "blue", os.Stderr)
	} else if strings.HasPrefix(text, "r+") {
		LogC(text[2:], "red", os.Stderr)
	} else if strings.HasPrefix(text, "--") {
		if IsDebugLow() {
			LogC(text[2:], "yellow", os.Stderr)
		}
	} else if strings.HasPrefix(text, "-") {
		if GetLogLevel() >= DebugLevel {
			// LogC(text[1:], "yellow", os.Stderr)
			localLogDbg.Msg(text[1:])
		}
	} else {
		// fmt.Fprintf(os.Stderr, "%s -- %s\n", time.Now().Format("2006-01-02 15:04:05"), text)
		localLogInf.Msg(text)
	}
}

func getCaller(start, level int) string {
	callStrArr := []string{}
	for i := start; i <= start+level-1; i++ {
		pc, file, no, ok := runtime.Caller(i)
		details := runtime.FuncForPC(pc)
		funcNameArr := strings.Split(details.Name(), ".")
		funcName := funcNameArr[len(funcNameArr)-1]
		if ok {
			fileArr := strings.Split(file, "/")
			callStr := F("%s:%d %s", fileArr[len(fileArr)-1], no, funcName)
			callStrArr = append(callStrArr, callStr)
		}
	}
	sort.SliceStable(callStrArr, func(i, j int) bool {
		return true
	})
	return strings.Join(callStrArr, " > ")
}

func addCaller(args []interface{}) []interface{} {
	callerStart := 3
	newArgs := []any{}
	for _, arg := range args {
		if a := cast.ToString(arg); strings.HasPrefix(a, "_DEBUG_CALLER_START=") {
			callerStart = cast.ToInt(strings.TrimPrefix(a, "_DEBUG_CALLER_START="))
		} else {
			newArgs = append(newArgs, arg)
		}
	}

	if CallerLevel == 0 {
		return newArgs
	}
	if caller := getCaller(callerStart, CallerLevel); caller != "" {
		newArgs = append(newArgs, M("caller", caller))
	}

	return newArgs
}

// Debug : print text in debug level
func Debug(text string, args ...interface{}) {
	args = addCaller(args)
	doHooks(zerolog.DebugLevel, text, args)
	doLog(ZLogErr.Debug(), text, args)
}

// DebugLow : print text in debug low level
func DebugLow(text string, args ...interface{}) {
	args = addCaller(args)
	doHooks(zerolog.DebugLevel, text, args)
	if IsDebugLow() {
		doLog(ZLogErr.Debug(), text, args)
	}
}

// Info : print text in info level
func Info(text string, args ...interface{}) {
	args = addCaller(args)
	doHooks(zerolog.InfoLevel, text, args)
	doLog(ZLogErr.Info(), text, args)
}

// Err : print text in error level
func Err(text string, args ...interface{}) {
	args = addCaller(args)
	doHooks(zerolog.ErrorLevel, text, args)
	doLog(ZLogErr.Error(), text, args)
}

func doHooks(level zerolog.Level, text string, args []interface{}) {
	for _, hook := range LogHooks {
		if level >= hook.Level && hook.Func != nil {
			hook.Func(&LogLine{
				Time:  time.Now(),
				Level: int8(level),
				Text:  text,
				Args:  args,
			})
		}
	}
}

func doLog(localLog *zerolog.Event, text string, args []interface{}) {
	args = extractLogMapArgs(args, localLog)
	text = F(text, args...)
	localLog.Msg(text)
}

// LogC : print text in specified color
func LogC(text string, col string, w io.Writer) {
	var textColored string

	switch col {
	case "red":
		textColored = Colorize(ColorRed, text)
	case "green":
		textColored = color.GreenString(text)
	case "blue":
		textColored = color.BlueString(text)
	case "magenta":
		textColored = color.MagentaString(text)
	case "white":
		textColored = color.WhiteString(text)
	case "cyan":
		textColored = color.CyanString(text)
	case "yellow":
		textColored = color.YellowString(text)
	default:
		textColored = text
	}

	if disableColor() {
		textColored = text
	}

	if runtime.GOOS == "windows" {
		w = color.Output
	}
	fmt.Fprintf(w, "%s -- %s\n", TimeColored(), textColored)
}

// PrintFatal prints the fatal error message
func PrintFatal(E error, args ...interface{}) {
	makeErrStrings := func(payload string) string {
		cancelledCount := 0
		errParts := strings.Split(payload, "\n\n")
		errStrings := []string{}
		errHash := map[string]struct{}{}
		for _, errPart := range errParts {
			if _, ok := errHash[errPart]; !ok && errPart != "context canceled" {
				if ps := strings.Split(errPart, "\n"); ps[len(ps)-1] == "context canceled" {
					cancelledCount++
				}
				errStrings = append(errStrings, errPart)
			}
			errHash[errPart] = struct{}{}
		}

		if cancelledCount == len(errStrings) {
			return "cancelled"
		}
		return strings.Join(errStrings, "\n\n")
	}

	prefix := "fatal:\n"
	if E != nil {
		err, ok := E.(*ErrType)
		if !ok {
			err = NewError(3, E, args...).(*ErrType)
		}

		eG, ok := E.(*ErrorGroup)
		if ok {
			if !IsDebugLow() {
				println(Colorize(ColorRed, prefix+eG.Error()))
			} else {
				println(Colorize(ColorRed, prefix+eG.Debug())) // stderr for detailed
			}
		} else {
			if !IsDebugLow() {
				joined := makeErrStrings(err.Error())
				println(Colorize(ColorRed, prefix+joined))
			} else {
				joined := makeErrStrings(err.Err)
				output := F("%s\n%s", strings.Join(err.Stack(), "\n"), joined)
				println(Colorize(ColorRed, prefix+output)) // stderr for detailed
			}
		}
	}
}

// deferOnFatal are cleanup function on fatal exit
var deferOnFatal = []func(){}

func DeferOnFatal(f func()) {
	deferOnFatal = append(deferOnFatal, f)
}

// LogFatal handles logging of an error and exits, useful for reporting
func LogFatal(E error, args ...interface{}) {
	if E != nil {
		PrintFatal(E, args...)

		for _, f := range deferOnFatal {
			f()
		}

		os.Exit(1)
	}
}

// Trace : print text in trace level
func Trace(text string, args ...interface{}) {
	args = addCaller(args)
	doHooks(zerolog.TraceLevel, text, args)
	doLog(ZLogErr.Trace(), text, args)
}

// Warn : print text in warning level
func Warn(text string, args ...interface{}) {
	args = addCaller(args)
	doHooks(zerolog.WarnLevel, text, args)
	doLog(ZLogErr.Warn(), text, args)
}

// TimeColored returns the time colored
func TimeColored() string {
	if disableColor() {
		return time.Now().Format("2006-01-02 15:04:05")
	}
	return color.CyanString(time.Now().Format("2006-01-02 15:04:05"))
}

func Colorize(color int, text string) string {
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", color, text)
}
