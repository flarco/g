package gutil

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"

	"github.com/markbates/pkger"
	"github.com/markbates/pkger/pkging"

	color "github.com/fatih/color"
	"github.com/flarco/gutil/stacktrace"
	"github.com/rs/zerolog"
	"github.com/spf13/cast"
	gomail "gopkg.in/gomail.v2"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
	// SMTPServer is email SMTP server host
	SMTPServer = "smtp.gmail.com"

	// SMTPPort is email SMTP server port
	SMTPPort = 465

	// SMTPUser is SMTP user name
	SMTPUser = os.Getenv("SLINGELT_SMTP_USER")

	// SMTPPass is user password
	SMTPPass = os.Getenv("SLINGELT_SMTP_PASSWORD")

	// AlertEmail is the email address to send errors to
	AlertEmail = os.Getenv("SLINGELT_ALERT_EMAIL")

	randSeeded = false

	logOut zerolog.Logger
	logErr zerolog.Logger
	// LogHooks are log hooks
	LogHooks    = []func(t string, a []interface{}){}
	callerLevel = 0
)

const (
	// AlphaRunes are alphabetic chars
	AlphaRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// AlphaRunesLower are lowercase alphabetic chars
	AlphaRunesLower = "abcdefghijklmnopqrstuvwxyz"
	// AlphaRunesUpper are uppercase alphabetic chars
	AlphaRunesUpper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// NumericRunes are numeric chars
	NumericRunes = "1234567890"
	// AplhanumericRunes are alphanumeric chars
	AplhanumericRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	// TokenRunes are alphanumeric+ chars for tokens
	TokenRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890_."
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if os.Getenv("SLINGELT_DEBUG_CALLER_LEVEL") != "" {
		callerLevel = cast.ToInt(os.Getenv("SLINGELT_DEBUG_CALLER_LEVEL"))
	}
	if os.Getenv("SLINGELT_DEBUG") == "TRACE" {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	} else if os.Getenv("SLINGELT_DEBUG") != "" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	outputOut := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05"}
	outputErr := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"}
	outputOut.FormatErrFieldValue = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}
	outputErr.FormatErrFieldValue = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}
	// if os.Getenv("ZLOG") != "PROD" {
	// 	zlog.Logger = zerolog.New(outputErr).With().Timestamp().Logger()
	// }

	if os.Getenv("SLINGELT_LOGGING") == "TASK" {
		outputOut.NoColor = true
		outputErr.NoColor = true
		logOut = zerolog.New(outputOut).With().Timestamp().Logger()
		logErr = zerolog.New(outputErr).With().Timestamp().Logger()
	} else if os.Getenv("SLINGELT_LOGGING") == "MASTER" || os.Getenv("SLINGELT_LOGGING") == "WORKER" {
		zerolog.LevelFieldName = "lvl"
		zerolog.MessageFieldName = "msg"
		logOut = zerolog.New(os.Stdout).With().Timestamp().Logger()
		logErr = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		outputErr = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "3:04PM"}
		if IsDebugLow() {
			outputErr = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"}
		}
		logOut = zerolog.New(outputErr).With().Timestamp().Logger()
		logErr = zerolog.New(outputErr).With().Timestamp().Logger()
	}
}

func disableColor() bool {
	return !cast.ToBool(os.Getenv("SLINGELT_LOGGING_COLOR"))
}

// SetZeroLogHook sets a zero log hook
func SetZeroLogHook(h zerolog.Hook) {
	logOut = logOut.Hook(h)
	logErr = logErr.Hook(h)
}

// SetLogHook sets a log hook
func SetLogHook(f func(t string, a []interface{})) {
	LogHooks = append(LogHooks, f)
}

// IsTask returns true is is TASK
func IsTask() bool {
	return os.Getenv("SLINGELT_LOGGING") == "TASK"
}

// IsDebugLow returns true is debug is low
func IsDebugLow() bool {
	return os.Getenv("SLINGELT_DEBUG") == "LOW" || os.Getenv("SLINGELT_DEBUG") == "TRACE"
}

// GetType : return the type of an interface
func GetType(myvar interface{}) string {
	t := reflect.TypeOf(myvar)
	if t.Kind() == reflect.Ptr {
		return "*" + t.Elem().Name()
	}
	return t.Name()
}

// F : fmt.Sprintf
func F(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

// M : return map[string]interface from args
func M(args ...interface{}) map[string]interface{} {
	mapInterf := map[string]interface{}{}
	key := ""
	for i, val := range args {
		if i%2 == 0 {
			key = cast.ToString(val)
		} else {
			mapInterf[key] = val
		}
	}
	return mapInterf
}

// ArrMapInt create a map from an array of integers
func ArrMapInt(args []int) map[int]interface{} {
	mapInterf := map[int]interface{}{}
	for _, val := range args {
		mapInterf[val] = val
	}
	return mapInterf
}

// ArrMapString create a map from an array of strings
func ArrMapString(args []string, toLower ...bool) map[string]interface{} {
	lower := false
	if len(toLower) > 0 && toLower[0] {
		lower = true
	}

	mapInterf := map[string]interface{}{}
	for _, val := range args {
		if lower {
			mapInterf[strings.ToLower(val)] = val
		} else {
			mapInterf[val] = val
		}
	}
	return mapInterf
}

// mapMerge copies key values from `mS` into `mT`
func mapMerge(mT map[string]interface{}, mS map[string]interface{}) map[string]interface{} {
	for key, val := range mS {
		mT[key] = val
	}
	return mT
}

// R : Replacer
// R("File {file} had error {error}", "file", file, "error", err)
func R(format string, args ...string) string {
	args2 := make([]string, len(args))
	for i, v := range args {
		if i%2 == 0 {
			args2[i] = fmt.Sprintf("{%v}", v)
		} else {
			args2[i] = fmt.Sprint(v)
		}
	}
	r := strings.NewReplacer(args2...)
	return r.Replace(format)
}

// Rm is like R, for replacing with a map
func Rm(format string, m map[string]interface{}) string {
	if m == nil {
		return format
	}

	args, i := make([]string, len(m)*2), 0
	for k, v := range m {
		args[i] = "{" + k + "}"
		args[i+1] = cast.ToString(v)
		i += 2
	}
	return strings.NewReplacer(args...).Replace(format)
}

// P prints the value of object
func P(v interface{}) {
	if IsDebugLow() {
		args := addCaller([]interface{}{})
		doLog(logErr.Debug(), F("%#v", v), args)
	}
}

// PrintT prints the type of object
func PrintT(v interface{}) {
	if IsDebugLow() {
		args := addCaller([]interface{}{})
		doLog(logErr.Debug(), F("%T", v), args)
	}
}

// PrintRows prints the rows of object
func PrintRows(rows [][]interface{}) {
	for _, row := range rows {
		P(row)
	}
}

// ErrStack is a modified version of stacktrace Propagate
func ErrStack(err error) error {
	return stacktrace.Propagate(err, "Error:", 3)
}

// IsErr : checks for error
func IsErr(err error, msg string) bool {
	if err != nil {
		LogError(stacktrace.Propagate(err, msg, 3))
		return true
	}
	return false
}

func isErrP(err error, msg string, callerSkip int) bool {
	if err != nil {
		LogError(stacktrace.Propagate(err, msg, callerSkip))
		return true
	}
	return false
}

// Now : Get unix epoch time in milli
func Now() int64 {
	return int64(time.Now().UnixNano() / 1000000)
}

// NowFileStr : Get millisecond time in file string format
func NowFileStr() string {
	return time.Now().Format("2006-01-02T150405.000")
}

func uintStr(val string) uint {
	val64, err := strconv.ParseUint(val, 10, 32)
	isErrP(err, "Failed to ParseUint", 4)
	return uint(val64)
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
	localLogDbg := logErr.Debug()
	localLogInf := logOut.Info()
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
		if os.Getenv("SLINGELT_DEBUG") != "" {
			// LogC(text[1:], "yellow", os.Stderr)
			localLogDbg.Msg(text[1:])
		}
	} else {
		// fmt.Fprintf(os.Stderr, "%s -- %s\n", time.Now().Format("2006-01-02 15:04:05"), text)
		localLogInf.Msg(text)
	}
}

func addCaller(args []interface{}) []interface{} {
	if callerLevel == 0 {
		return args
	}
	callStrArr := []string{}
	for i := 2; i <= 2+callerLevel-1; i++ {
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
	if len(callStrArr) > 0 {
		sort.SliceStable(callStrArr, func(i, j int) bool {
			return true
		})
		args = append(args, M("caller", strings.Join(callStrArr, " > ")))
	}
	return args
}

// Debug : print text in debug level
func Debug(text string, args ...interface{}) {
	args = addCaller(args)
	doHooks(zerolog.DebugLevel, text, args)
	doLog(logErr.Debug(), text, args)
}

// Info : print text in info level
func Info(text string, args ...interface{}) {
	doHooks(zerolog.InfoLevel, text, args)
	if IsTask() {
		doLog(logOut.Info(), text, args)
	} else {
		doLog(logErr.Info(), text, args)
	}
}

// Trace : print text in trace level
func Trace(text string, args ...interface{}) {
	args = addCaller(args)
	doHooks(zerolog.TraceLevel, text, args)
	doLog(logErr.Trace(), text, args)
}

// Warn : print text in warning level
func Warn(text string, args ...interface{}) {
	doHooks(zerolog.WarnLevel, text, args)
	if IsTask() {
		doLog(logOut.Warn(), text, args)
	} else {
		doLog(logErr.Warn(), text, args)
	}
}

func doHooks(level zerolog.Level, text string, args []interface{}) {
	if zerolog.GlobalLevel() == level {
		for _, hook := range LogHooks {
			hook(text, args)
		}
	}
}

func doLog(localLog *zerolog.Event, text string, args []interface{}) {
	args = extractLogMapArgs(args, localLog)
	text = F(text, args...)
	localLog.Msg(text)
}

// TimeColored returns the time colored
func TimeColored() string {
	if disableColor() {
		return time.Now().Format("2006-01-02 15:04:05")
	}
	return color.CyanString(time.Now().Format("2006-01-02 15:04:05"))
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

// LogError handles logging of an error, useful for reporting
func LogError(E error, args ...interface{}) {
	msg := ArgsErrMsg(args...)
	if E != nil {
		if !strings.Contains(E.Error(), " --- at ") && IsDebugLow() {
			E = stacktrace.Propagate(E, "error:", 3) // add stack
		}
		doHooks(zerolog.DebugLevel, E.Error(), args)
		if IsTask() {
			simpleErr := errors.New(ErrMsgSimple(E))
			logOut.Err(simpleErr).Msg(msg) // simple message in STDOUT
		}
		logErr.Err(E).Msg(msg) // detailed error in STDERR
	}
}

// LogFatal handles logging of an error and exits, useful for reporting
func LogFatal(E error, args ...interface{}) {
	if E != nil {

		if !IsDebugLow() {
			println(color.RedString(ErrMsgSimple(E)))
			os.Exit(1)
		}

		if !strings.Contains(E.Error(), " --- at ") {
			E = stacktrace.Propagate(E, "error:", 3) // add stack
		}

		if IsTask() {
			fmt.Fprintf(os.Stdout, ErrMsgSimple(E)) // stdout simple err
		}
		println(color.RedString(E.Error())) // stderr for detailed
		os.Exit(1)
	}
}

// ArgsErrMsg takes args and makes an error message
func ArgsErrMsg(args ...interface{}) (msg string) {
	if len(args) == 1 {
		msg = cast.ToString(args[0])
	} else if len(args) > 1 {
		msg = F(cast.ToString(args[0]), args[1:]...)
	} else {
		msg = "error:"
	}
	return
}

//ErrMsg returns a simple error message
func ErrMsg(e error) string {
	if e == nil {
		return ""
	}
	msgLines := strings.Split(e.Error(), "\n")
	msgArr := []string{}
	for _, line := range msgLines {
		if !strings.HasPrefix(line, " --- at ") && line != "error:" {
			msgArr = append(msgArr, line)
		}
	}
	return strings.Join(msgArr, "\n")
	// msgArr = strings.Split(e.Error(), `Caused by: `)
	// msg := msgArr[len(msgArr)-1]
	// return msg
}

// ErrMsgSimple returns a simpler error message
func ErrMsgSimple(e error) string {
	if e == nil {
		return ""
	}
	msgLines := strings.Split(e.Error(), "\n")
	msgArr := []string{}
	currErrMsg := []string{}
	for _, line := range msgLines {
		if strings.HasPrefix(line, "~ ") {
			msgArr = append(msgArr, strings.Join(currErrMsg, "\n"))
			currErrMsg = []string{strings.TrimPrefix(line, "~ ")}
			continue
		} else if strings.HasPrefix(line, " --- at") {
			msgArr = append(msgArr, strings.Join(currErrMsg, "\n"))
			currErrMsg = []string{strings.TrimPrefix(line, " --- at")}
			continue
		} else if line != "error:" && strings.TrimSpace(line) != "" {
			currErrMsg = append(currErrMsg, line)
		}
	}
	msgArr = append(msgArr, strings.Join(currErrMsg, "\n")) // last err
	msg := msgArr[len(msgArr)-1]
	return msg
}

// ErrorText returns the error text if error is not nul
func ErrorText(e error) string {
	if e != nil {
		return e.Error()
	}
	return ""
}

// Error returns stacktrace error with message
func Error(e error, args ...interface{}) error {
	msg := ArgsErrMsg(args...)

	if e != nil {
		doHooks(zerolog.DebugLevel, F("%s ~ %s", msg, e.Error()), []interface{}{})
		if IsDebugLow() {
			return stacktrace.Propagate(e, msg, 3)
		}
		return fmt.Errorf("~ %s\n%s", msg, e.Error())
	}

	err := fmt.Errorf("err is nil! Need to add if err != nil")
	logErr.Err(stacktrace.Propagate(err, msg, 3)).Msg("err is nil! Need to add if err != nil")

	return nil
}

// LogErrorMail handles logging of an error and mail it to self
func LogErrorMail(E error) {
	LogCRedErr(E.Error())
	SendMail(SMTPUser, []string{AlertEmail}, "Error | "+os.Args[0], E.Error())
}

// LogIfError handles logging of an error if it i not nil, useful for reporting
func LogIfError(E error) {
	if E != nil {
		LogError(E)
	}
}

// SendMail sends an email to the specific email address
// https://godoc.org/gopkg.in/gomail.v2#example-package
func SendMail(from string, to []string, subject string, textHTML string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to...)
	// m.SetAddressHeader("Cc", "dan@example.com", "Dan")
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", textHTML)
	// m.Attach("/home/Alex/lolcat.jpg")

	d := gomail.NewDialer(SMTPServer, SMTPPort, SMTPUser, SMTPPass)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	// Send the email
	err := d.DialAndSend(m)
	return err
}

// Tee prints stream of text of reader
func Tee(reader io.Reader, limit int) io.Reader {
	pipeR, pipeW := io.Pipe()

	cnt := 0
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			cnt++
			if cnt > limit {
				break
			}
			bytes := scanner.Bytes()
			nl := []byte("\n")
			fmt.Println(string(bytes))
			pipeW.Write(append(bytes, nl...))
		}
		pipeW.Close()
	}()

	return pipeR
}

// RandString returns a random string of len n with the provided char set
// charset can be `AlphaRunes`, `AlphaRunesLower`, `AlphaRunesUpper` or `AplhanumericRunes`
func RandString(charset string, n int) string {
	if !randSeeded {
		rand.Seed(time.Now().UnixNano())
		randSeeded = true
	}
	b := make([]byte, n)

	for i := range b {
		b[i] = charset[rand.Int63()%int64(len(charset))]
	}

	return string(b)
}

// RandInt64 returns a random positive number up to max
func RandInt64(max int64) int64 {
	if !randSeeded {
		rand.Seed(time.Now().UnixNano())
		randSeeded = true
	}
	return rand.Int63n(max)
}

// RandInt returns a random positive number up to max
func RandInt(max int) int {
	if !randSeeded {
		rand.Seed(time.Now().UnixNano())
		randSeeded = true
	}
	return rand.Intn(max)
}

// DownloadFile downloads a file
func DownloadFile(url string, filepath string) (err error) {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return Error(err, "Unable to Create file "+filepath)
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return Error(err, "Unable to Reach URL: "+url)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return Error(err, F("Bad Status '%s' from URL %s", resp.Status, url))
	}

	// Writer the body to file
	bw, err := io.Copy(out, resp.Body)
	if err != nil || bw == 0 {
		return Error(err, "Unable to write to file "+filepath)
	}

	return nil
}

// KVArrToMap parse a Key-Value array in the form of
// `"Prop1=Value1", "Prop2=Value2", ...`
// and return a map
func KVArrToMap(props ...string) map[string]string {
	properties := map[string]string{}
	for _, propStr := range props {
		arr := strings.Split(propStr, "=")
		if len(arr) == 1 && arr[0] != "" {
			properties[arr[0]] = ""
		} else if len(arr) == 2 {
			properties[arr[0]] = arr[1]
		} else if len(arr) > 2 {
			val := strings.Join(arr[1:], "=")
			properties[arr[0]] = val
		}
	}
	return properties
}

// MapToKVArr transforms a map into a key-value array
// such as: `"Prop1=Value1", "Prop2=Value2", ...`
func MapToKVArr(properties map[string]string) []string {
	props := []string{}
	for k, v := range properties {
		props = append(props, F("%s=%s", k, v))
	}
	return props
}

// ErrorGroup represents a group of errors
type ErrorGroup struct {
	Logging bool
	Errors  []error
}

// Add adds an error to the error group
func (e *ErrorGroup) Add(err error) {
	e.Errors = append(e.Errors, err)
}

// Len returns the number of errors captured
func (e *ErrorGroup) Len() int {
	return len(e.Errors)
}

// Capture adds an error to the error group, and return true if err was not nil
func (e *ErrorGroup) Capture(err error) bool {
	if err != nil {
		e.Errors = append(e.Errors, err)
		if e.Logging || IsDebugLow() {
			LogError(err)
		}
		return true
	}
	return false
}

// Reset reset the errors to none
func (e *ErrorGroup) Reset() {
	e.Errors = []error{}
}

// Err returns an error if any errors had been added
func (e *ErrorGroup) Err() error {
	if len(e.Errors) == 0 {
		return nil
	}

	var errstrings []string
	for _, er := range e.Errors {
		errstrings = append(errstrings, er.Error())
	}
	return fmt.Errorf(strings.Join(errstrings, "\n"))
}

func Must(e error) {
	if e != nil {
		panic(stacktrace.Propagate(e, "", 3))
	}
}

// ErrJSON returns to the echo.Context a formatted
func ErrJSON(HTTPStatus int, err error, args ...interface{}) error {
	msg := ArgsErrMsg(args...)
	LogError(stacktrace.Propagate(err, msg, 3))
	return echo.NewHTTPError(HTTPStatus, M("message", msg, "error", ErrMsg(err)))
}

// GetPort asks the kernel for a free open port that is ready to use.
func GetPort(hostPort string) (int, error) {
	if hostPort == "" {
		hostPort = "localhost:0"
	}
	addr, err := net.ResolveTCPAddr("tcp", hostPort)
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// UserHomeDir returns the home directory of the running user
func UserHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	} else if runtime.GOOS == "linux" {
		home := os.Getenv("XDG_CONFIG_HOME")
		if home != "" {
			return home
		}
	}
	return os.Getenv("HOME")
}

// Peek allows peeking without advancing the reader
func Peek(reader io.Reader, n int) (data []byte, readerNew io.Reader, err error) {
	bReader := bufio.NewReader(reader)
	if n == 0 {
		n = bReader.Size()
	}
	readerNew = bReader
	data, err = bReader.Peek(n)
	if err == io.EOF {
		err = nil
	} else if err != nil {
		err = Error(err, "could not Peek")
		return
	}

	return
}

// PkgerString returns the packager file string
func PkgerString(name string) (content string, err error) {
	_, filename, _, _ := runtime.Caller(1)
	file, err := pkger.Open(path.Join(path.Dir(filename), name))
	if err != nil {
		return "", Error(err, "could not open pker file: %s", name)
	}

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", Error(err, "could not read pker file: ", name)
	}
	return string(fileBytes), nil
}

// PkgerFile returns the packager file
func PkgerFile(name string) (file pkging.File, err error) {
	_, filename, _, _ := runtime.Caller(1)
	TemplateFile, err := pkger.Open(path.Join(path.Dir(filename), name))
	if err != nil {
		return nil, Error(err, "cannot open "+path.Join(path.Dir(filename), name))
	}
	return TemplateFile, nil
}

// ClientDoStream Http client method execution returning a reader
func ClientDoStream(method, URL string, body io.Reader, headers map[string]string) (resp *http.Response, reader io.Reader, err error) {
	Trace("%s -> %s", method, URL)
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, nil, Error(err, "could not %s @ %s", method, URL)
	}
	if headers == nil {
		headers = map[string]string{}
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := http.Client{}

	resp, err = client.Do(req)
	if err != nil {
		err = Error(err, "could not perform request")
		return
	}

	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		respBytes, _ := ioutil.ReadAll(resp.Body)
		err = Error(fmt.Errorf("Unexpected Response %d: %s. %s", resp.StatusCode, resp.Status, string(respBytes)))
		return
	}

	reader = bufio.NewReader(resp.Body)

	return
}

// ClientDo Http client method execution
func ClientDo(method, URL string, body io.Reader, headers map[string]string, timeOut ...int) (resp *http.Response, respBytes []byte, err error) {
	to := 3600 * time.Second
	if len(timeOut) > 0 {
		to = time.Duration(timeOut[0]) * time.Second
	}

	Trace("%s -> %s", method, URL)
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, nil, Error(err, "could not %s @ %s", method, URL)
	}
	if headers == nil {
		headers = map[string]string{}
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := http.Client{Timeout: to}

	resp, err = client.Do(req)
	if err != nil {
		err = Error(err, "could not perform request")
		return
	}

	respBytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		err = Error(err, "could not read from request body")
	}

	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		err = Error(fmt.Errorf("Unexpected Response %d: %s. %s", resp.StatusCode, resp.Status, string(respBytes)))
		return
	}

	return
}

// MJ returns the JSON string of a map
func MJ(args ...interface{}) string {
	return string(MarshalMap(M(args...)))
}

// MarshalMap marshals a map into json
func MarshalMap(m map[string]interface{}) []byte {
	jBytes, _ := json.Marshal(m)
	return jBytes
}

// UnmarshalMap unmarshals into a map of interface
func UnmarshalMap(s string) (map[string]interface{}, error) {
	m := M()
	err := json.Unmarshal([]byte(s), &m)
	if err != nil {
		err = Error(err, "could not unmarshal into map")
		return m, err
	}
	return m, nil
}

// NewTsID creates a new timestamp ID
func NewTsID(prefix ...string) string {
	p := ""
	if len(prefix) > 0 {
		p = prefix[0] + "."
	}
	tsMilli := int64(cast.ToFloat64(time.Now().UnixNano()) / 1000000.0)
	return F("%s%d.%s", p, tsMilli, RandString(AplhanumericRunes, 3))
}

// ArrI returns an array of interface
func ArrI(items ...interface{}) []interface{} {
	return items
}

// ArrStr returns an array of strings
func ArrStr(items ...string) []string {
	return items
}

// ArrContains returns true if array of strings contains
func ArrContains(items []string, subItem string) bool {
	_, ok := ArrMapString(items)[subItem]
	return ok
}

// StructField is a field of a struct
type StructField struct {
	Field reflect.StructField
	Value reflect.Value
	JKey  string
}

// StructFields returns the fields of a struct
func StructFields(obj interface{}) (fields []StructField) {
	var t reflect.Type
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		t = reflect.Indirect(value).Type()
	} else {
		t = reflect.TypeOf(obj)
	}
	for i := 0; i < t.NumField(); i++ {
		var valueField reflect.Value
		if value.Kind() == reflect.Ptr {
			valueField = value.Elem().Field(i)
		} else {
			valueField = value.Field(i)
		}
		sField := t.Field(i)
		jKey := strings.Split(sField.Tag.Get("json"), ",")[0]
		fields = append(fields, StructField{sField, valueField, jKey})
	}
	return fields
}

// CloneValue clones a pointer to another
func CloneValue(source interface{}, destin interface{}) {
	x := reflect.ValueOf(source)
	if x.Kind() == reflect.Ptr {
		starX := x.Elem()
		y := reflect.New(starX.Type())
		starY := y.Elem()
		starY.Set(starX)
		reflect.ValueOf(destin).Elem().Set(y.Elem())
	} else {
		destin = x.Interface()
	}
}

// LogEvent logs to Graylog
func LogEvent(m map[string]interface{}) {
	if os.Getenv("SLINGELT_SEND_ANON_USAGE") != "" {
		return
	}

	URL := "https://logapi.slingelt.com/log/event/prd"
	if os.Getenv("SLINGELT_ENV") == "STG" {
		URL = "https://logapi.slingelt.com/log/event/stg"
	}

	jsonBytes, err := json.Marshal(m)
	if err != nil {
		if IsDebugLow() {
			LogError(err)
		}
	}

	_, _, err = ClientDo(
		"POST",
		URL,
		bytes.NewBuffer(jsonBytes),
		map[string]string{"Content-Type": "application/json"},
		1,
	)

	if err != nil {
		if IsDebugLow() {
			LogError(err)
		}
	}
}
