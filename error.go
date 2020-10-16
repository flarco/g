package gutil

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/flarco/gutil/stacktrace"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/spf13/cast"
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
func (e *ErrType) ErrorFull() string {
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

// DebugError returns an error type with a detailed string
func (e *ErrType) DebugError() error {
	return fmt.Errorf(e.Debug())
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

func getCallerStack() []string {
	callerArr := []string{}
	i := 2
	for {
		pc, file, no, ok := runtime.Caller(i)
		if !ok {
			break
		}
		details := runtime.FuncForPC(pc)
		funcNameArr := strings.Split(details.Name(), ".")
		funcName := funcNameArr[len(funcNameArr)-1]
		fileArr := strings.Split(file, "/")
		callStr := F("%s:%d %s", fileArr[len(fileArr)-1], no, funcName)
		callerArr = append(callerArr, callStr)
		i++
	}
	return callerArr
}

// NewErrorType returns an Errtype error
func NewErrorType(e interface{}, args ...interface{}) *ErrType {
	if e == nil {
		return nil
	}

	MsgStack := []string{ArgsErrMsg(args...)}
	Err := cast.ToString(e)
	CallerStack := getCallerStack()
	Position := 0

	switch e.(type) {
	case *ErrType:
		errPrev := e.(*ErrType)
		Err = errPrev.Err
		MsgStack = append(errPrev.MsgStack, MsgStack...)
		CallerStack = append([]string{errPrev.CallerStack[0]}, CallerStack...)
		Position = errPrev.Position + 1
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

	if e == nil {
		return nil
	}

	MsgStack := []string{ArgsErrMsg(args...)}
	Err := cast.ToString(e)
	CallerStack := getCallerStack()
	Position := 0

	switch e.(type) {
	case *ErrType:
		errPrev := e.(*ErrType)
		Err = errPrev.Err
		MsgStack = append(errPrev.MsgStack, MsgStack...)
		CallerStack = append([]string{errPrev.CallerStack[0]}, CallerStack...)
		Position = errPrev.Position + 1
	}

	return &ErrType{
		Err:         Err,
		MsgStack:    MsgStack,
		CallerStack: CallerStack,
		Position:    Position,
	}
}

// ErrorOld returns stacktrace error with message
func ErrorOld(e error, args ...interface{}) error {
	msg := ArgsErrMsg(args...)

	if e != nil {
		// doHooks(zerolog.DebugLevel, F("%s ~ %s", msg, e.Error()), []interface{}{})
		if IsDebugLow() {
			return stacktrace.Propagate(e, msg, 3)
		}
		return fmt.Errorf("~ %s\n%s", msg, e.Error())
	}

	err := fmt.Errorf("err is nil! Need to add if err != nil")
	LogErr.Err(stacktrace.Propagate(err, msg, 3)).Msg("err is nil! Need to add if err != nil")

	return nil
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

// LogError handles logging of an error, useful for reporting
func LogError(E error, args ...interface{}) {
	if E != nil {
		msg := ArgsErrMsg(args...)
		doHooks(zerolog.DebugLevel, E.Error(), args)
		if IsTask() {
			simpleErr := errors.New(ErrMsgSimple(E))
			LogOut.Err(simpleErr).Msg(msg) // simple message in STDOUT
		}
		err, ok := E.(*ErrType)
		if ok {
			E = err.DebugError()
		}
		LogErr.Err(E).Msg(msg) // detailed error in STDERR
	}
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

	errstrings := []string{}
	for _, er := range e.Errors {
		errstrings = append(errstrings, er.Error())
	}
	return fmt.Errorf(strings.Join(errstrings, "\n"))
}

// ErrJSON returns to the echo.Context a formatted
func ErrJSON(HTTPStatus int, err error, args ...interface{}) error {
	msg := ArgsErrMsg(args...)
	LogError(err)
	return echo.NewHTTPError(HTTPStatus, M("message", msg, "error", ErrMsg(err)))
}
