package g

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/rs/zerolog"
	"github.com/spf13/cast"
)

// LogHook is a hook to be perform at the specified level
type LogHook struct {
	Level    zerolog.Level
	Send     func(t string, a ...interface{})
	batch    [][]interface{}
	labels   map[string]string
	queue    chan lokiLine
	lastSent time.Time
	mux      sync.Mutex
	ticker   *time.Ticker
}

// LogHooks are log hooks
var LogHooks = []*LogHook{}

// LogLevel is the log level
var LogLevel = new(Level)

// ZLogOut is the non-error/normal logger
var ZLogOut = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05"}).With().Timestamp().Logger()

// ZLogErr is the error/debug logger
var ZLogErr = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"}).With().Timestamp().Logger()

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

// NewLogHook return a new log hook
func NewLogHook(level Level, doFunc func(t string, a ...interface{})) *LogHook {
	zLevel := zerolog.InfoLevel
	switch level {
	case TraceLevel:
		zLevel = zerolog.TraceLevel
	case DebugLevel, LowDebugLevel:
		zLevel = zerolog.DebugLevel
	case WarnLevel:
		zLevel = zerolog.WarnLevel
	}
	if doFunc == nil {
		doFunc = func(t string, a ...interface{}) {}
	}
	return &LogHook{
		Level:  zLevel,
		Send:   doFunc,
		queue:  make(chan lokiLine, 100000),
		batch:  [][]interface{}{},
		labels: map[string]string{},
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

// IsDebugLow returns true is debug is low
func IsDebugLow() bool {
	return GetLogLevel() == LowDebugLevel || GetLogLevel() == TraceLevel
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
	vv, _ := json.MarshalIndent(v, "", "  ")
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
	if CallerLevel == 0 {
		return args
	}

	if caller := getCaller(3, CallerLevel); caller != "" {
		args = append(args, M("caller", caller))
	}

	return args
}

// Debug : print text in debug level
func Debug(text string, args ...interface{}) {
	args = addCaller(args)
	doHooks(zerolog.DebugLevel, text, args)
	doLog(ZLogErr.Debug(), text, args)
}

// DebugLow : print text in debug low level
func DebugLow(text string, args ...interface{}) {
	if IsDebugLow() {
		args = addCaller(args)
		doHooks(zerolog.DebugLevel, text, args)
		doLog(ZLogErr.Debug(), text, args)
	}
}

// Info : print text in info level
func Info(text string, args ...interface{}) {
	doHooks(zerolog.InfoLevel, text, args)
	if IsTask() {
		doLog(ZLogOut.Info(), text, args)
	} else {
		doLog(ZLogErr.Info(), text, args)
	}
}

func doHooks(level zerolog.Level, text string, args []interface{}) {
	for _, hook := range LogHooks {
		if zerolog.GlobalLevel() >= hook.Level && hook.Send != nil {
			args = append(args, M("level", level.String()))
			go hook.Send(text, args...)
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
		textColored = color.RedString(text)
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

// LogCGreen prints in green
func LogCGreen(text string) { LogC(text, "green", os.Stderr) }

// LogCRed prints in red
func LogCRed(text string) { LogC(text, "red", os.Stderr) }

// LogCRedErr prints in red to Stderr
func LogCRedErr(text string) { LogC(text, "red", os.Stderr) }

// LogCBlue prints in blue
func LogCBlue(text string) { LogC(text, "blue", os.Stderr) }

// LogCMagenta print in magenta
func LogCMagenta(text string) { LogC(text, "magenta", os.Stderr) }

// LogCWhite prints in white
func LogCWhite(text string) { LogC(text, "white", os.Stderr) }

// LogCCyan prints in white
func LogCCyan(text string) { LogC(text, "cyan", os.Stderr) }

// LogFatal handles logging of an error and exits, useful for reporting
func LogFatal(E error, args ...interface{}) {
	prefix := "fatal:\n"
	if E != nil {
		err, ok := E.(*ErrType)
		if !ok {
			err = NewError(3, E, args...).(*ErrType)
		}

		if !IsDebugLow() {
			println(color.RedString(prefix + err.Full()))
			os.Exit(1)
		}

		if IsTask() {
			fmt.Fprintf(os.Stdout, err.Err) // stdout simple err
		}
		println(color.RedString(prefix + err.Debug())) // stderr for detailed
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
	doHooks(zerolog.WarnLevel, text, args)
	if IsTask() {
		doLog(ZLogOut.Warn(), text, args)
	} else {
		doLog(ZLogErr.Warn(), text, args)
	}
}

// TimeColored returns the time colored
func TimeColored() string {
	if disableColor() {
		return time.Now().Format("2006-01-02 15:04:05")
	}
	return color.CyanString(time.Now().Format("2006-01-02 15:04:05"))
}
