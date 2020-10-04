package gutil

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/flarco/gutil/stacktrace"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/spf13/cast"
)

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
			LogOut.Err(simpleErr).Msg(msg) // simple message in STDOUT
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
	LogError(stacktrace.Propagate(err, msg, 3))
	return echo.NewHTTPError(HTTPStatus, M("message", msg, "error", ErrMsg(err)))
}
