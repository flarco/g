package g

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
)

// ErrType is an error with details
type ErrType struct {
	Err         string   // the original error string
	MsgStack    []string // addition details for error context
	CallerStack []string // the caller stack
	Position    int      // the position in the array stack (0 is first)
}

// MsgStacked return a stacked error message
func (e *ErrType) MsgStacked() (m string) {
	ErrStack := append([]string{e.Err}, e.MsgStack...)
	return strings.Join(ErrStack, "\n")
}

// ErrorFull returns the full error stack
func (e *ErrType) Full() string {
	if len(e.MsgStack) == 0 {
		return e.Err
	}

	MsgStack := make([]string, len(e.MsgStack))
	copy(MsgStack, e.MsgStack)
	for i, j := 0, len(MsgStack)-1; i < j; i, j = i+1, j-1 {
		MsgStack[i], MsgStack[j] = MsgStack[j], MsgStack[i]
	}
	return F("~ %s\n%s", strings.Join(MsgStack, "\n~ "), e.Err)
}

func (e *ErrType) Error() string {
	if len(e.MsgStack) == 0 {
		return e.Err
	}
	return F("~ %s\n%s", e.MsgStack[0], e.Err)
}

// FullError returns an error type with a detailed string
func (e *ErrType) FullError() error {
	return fmt.Errorf(e.Full())
}

// DebugError returns an error type with a detailed string
func (e *ErrType) DebugError() error {
	return fmt.Errorf(e.Debug())
}

// OriginalError returns an error type with the original err string
func (e *ErrType) OriginalError() error {
	return fmt.Errorf(e.Err)
}

// Debug returns a stacked error for debugging
func (e *ErrType) Debug() string {
	if len(e.CallerStack) == 0 {
		return e.Err
	}

	stack := []string{}
	for i, caller := range e.CallerStack {
		msg := ""
		if len(e.MsgStack) > i {
			msg = e.MsgStack[i]
		}

		msgDbg := F("~ %s\n--- %s ---", msg, caller)
		if msg == "" {
			msgDbg = F("--- %s ---", caller)
		}
		stack = append(stack, msgDbg)
	}

	for i, j := 0, len(stack)-1; i < j; i, j = i+1, j-1 {
		stack[i], stack[j] = stack[j], stack[i]
	}
	return F("%s\n%s", strings.Join(stack, "\n"), e.Err)
}

func getCallerStack(levelsUp int) []string {
	callerArr := []string{}
	for {
		pc, file, no, ok := runtime.Caller(levelsUp)
		if !ok {
			break
		}
		details := runtime.FuncForPC(pc)
		funcNameArr := strings.Split(details.Name(), ".")
		funcName := funcNameArr[len(funcNameArr)-1]
		fileArr := strings.Split(file, "/")
		callStr := F("%s:%d %s", fileArr[len(fileArr)-1], no, funcName)
		if strings.Contains(callStr, "goexit") {
			break
		}
		callerArr = append(callerArr, callStr)
		levelsUp++
	}
	return callerArr
}

// NewError returns stacktrace error with message
func NewError(levelsUp int, e interface{}, args ...interface{}) error {

	CallerStack := getCallerStack(levelsUp)
	if e == nil {
		Warn("NewError called with nil error:\n  " + strings.Join(CallerStack, "\n  "))
		return nil
	}

	MsgStack := []string{ArgsErrMsg(args...)}
	Err := ""
	Position := 0

	errPrev := ErrType{}
	switch et := e.(type) {
	case *ErrType:
		Err = et.Err
		MsgStack = append(et.MsgStack, MsgStack...)
		CallerStack = et.CallerStack
		Position = et.Position + 1
	case string:
		if e0 := JSONUnmarshal([]byte(et), &errPrev); e0 == nil && len(errPrev.CallerStack) != 0 { // compatible with original flarco/g.Error
			Err = errPrev.Err
			MsgStack = append(errPrev.MsgStack, MsgStack...)
			Position = errPrev.Position + 1
			if CallerStack[0] == errPrev.CallerStack[0] {
				CallerStack = errPrev.CallerStack
			} else {
				CallerStack = append(errPrev.CallerStack, CallerStack...)
			}
		} else {
			MsgStack = []string{}
			Err = ArgsErrMsg(append([]any{et}, args...)...)
		}
	default:
		if e0 := JSONConvert(e, &errPrev); e0 == nil && len(errPrev.CallerStack) != 0 { // compatible with original flarco/g.Error
			Err = errPrev.Err
			MsgStack = append(errPrev.MsgStack, MsgStack...)
			Position = errPrev.Position + 1
			if CallerStack[0] == errPrev.CallerStack[0] {
				CallerStack = errPrev.CallerStack
			} else {
				CallerStack = append(errPrev.CallerStack, CallerStack...)
			}
		} else {
			MsgStack = []string{}
			switch et := e.(type) {
			case error:
				Err = ArgsErrMsg(append([]any{et.Error()}, args...)...)
			default:
				Err = ArgsErrMsg(append([]any{e}, args...)...)
			}
		}
	}

	return &ErrType{
		Err:         Err,
		MsgStack:    MsgStack,
		CallerStack: CallerStack,
		Position:    Position,
	}
}

// Error returns stacktrace error with message
func Error(e interface{}, args ...interface{}) error {
	return NewError(3, e, args...)
}

// ErrorIf allows use of `ErrorIf(err)` without the `if err != nil `
var ErrorIf = Error

// LogError handles logging of an error, useful for reporting
func LogError(E error, args ...interface{}) bool {
	if E != nil {
		msg := ArgsErrMsg(args...)
		err, ok := E.(*ErrType)
		if !ok {
			err = NewError(3, E, args...).(*ErrType)
		}
		doHooks(zerolog.DebugLevel, err.Error(), args)
		if IsTask() {
			LogOut.Err(err.OriginalError()).Msg(msg) // simple message in STDOUT
			LogErr.Err(err.DebugError()).Msg(msg)
		} else if IsDebugLow() {
			LogErr.Err(err.DebugError()).Msg(msg)
		} else {
			LogErr.Err(err.OriginalError()).Msg(msg) // detailed error in STDERR
		}
		return true
	}
	return false
}

// ArgsErrMsg takes args and makes an error message
func ArgsErrMsg(args ...interface{}) (msg string) {
	if len(args) == 1 {
		msg = cast.ToString(args[0])
	} else if len(args) > 1 {
		msg = F(cast.ToString(args[0]), args[1:]...)
	} else {
		msg = ""
	}
	return
}

// ErrContains returns true if the sub-tring is found in error string
func ErrContains(e error, subStr string) bool {
	if e == nil {
		return false
	}
	return strings.Contains(e.Error(), subStr)
}

//ErrMsg returns a simple error message
func ErrMsg(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}

// ErrMsgSimple returns a simpler error message
func ErrMsgSimple(e error) string {
	if e == nil {
		return ""
	}

	err, ok := e.(*ErrType)
	if !ok {
		return e.Error()
	}
	return err.Err
}

// ErrorText returns the error text if error is not nul
func ErrorText(e error) string {
	if e != nil {
		return e.Error()
	}
	return ""
}

// AssertNoError asserts there is no error an logs it if there is
func AssertNoError(t *testing.T, e error) bool {
	if e != nil {
		LogError(e)
	}
	return assert.NoError(t, e)
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

	errstrings := []string{}
	for _, err := range e.Errors {
		if err2, ok := err.(*ErrType); ok && IsDebugLow() {
			errstrings = append(errstrings, err2.Debug())
		} else {
			errstrings = append(errstrings, err.Error())
		}
	}
	return fmt.Errorf(strings.Join(errstrings, "\n"))
}

// ErrJSON returns to the echo.Context a formatted
func ErrJSON(HTTPStatus int, err error, args ...interface{}) error {
	msg := ArgsErrMsg(args...)
	LogError(err)
	if msg == "" {
		msg = ErrMsg(err)
	} else if ErrMsg(err) != "" {
		msg = F("%s [%s]", msg, ErrMsg(err))
	}
	return echo.NewHTTPError(HTTPStatus, M("error", msg))
}
