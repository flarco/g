package g

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
)

func init() {
	if SentryDsn != "" {
		SentryInit()
	}
}

// ErrType is an error with details
type ErrType struct {
	Err         string   // the original error string
	MsgStack    []string // addition details for error context
	CallerStack []string // the caller stack
	Position    int      // the position in the array stack (0 is first)
}

type ErrorInterface interface {
	String() string
	Debug() string
	MD5() string
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

	MsgStack := []string{}
	for _, msg := range e.MsgStack {
		if msg == "" {
			continue
		}
		MsgStack = append(MsgStack, msg)
	}

	for i, j := 0, len(MsgStack)-1; i < j; i, j = i+1, j-1 {
		MsgStack[i], MsgStack[j] = MsgStack[j], MsgStack[i]
	}
	return F("~ %s\n%s", strings.Join(MsgStack, "\n~ "), e.Err)
}

func (e *ErrType) Error() string {
	if len(e.MsgStack) == 0 || e.MsgStack[0] == "" {
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

// CallerStackMD5 returns the stack md5
func (e *ErrType) CallerStackMD5() string {
	return MD5(e.CallerStack...)
}

// LastCaller returns the last caller
func (e *ErrType) LastCaller() string {
	stack := e.Stack()
	if len(stack) == 0 {
		return "Unknown"
	}

	for _, caller := range e.CallerStack {
		return caller
	}

	return "Unknown"
}

// Stack returns the stack
func (e *ErrType) Stack() []string {
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
	return stack
}

func (e *ErrType) MD5() string {
	errHash := MD5(e.Err)
	if len(e.CallerStack) > 0 {
		errHash = MD5(e.CallerStack[0], e.Err)
	}
	return errHash
}

// Debug returns a stacked error for debugging
func (e *ErrType) Debug() string {
	if len(e.CallerStack) == 0 {
		return e.Err
	}

	return F("%s\n%s", strings.Join(e.Stack(), "\n"), e.Err)
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
		if e0 := json.Unmarshal([]byte(et), &errPrev); e0 == nil && len(errPrev.CallerStack) != 0 { // compatible with original flarco/g.Error
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
				Err = et.Error()
				if len(args) > 0 {
					MsgStack = []string{ArgsErrMsg(args...)}
				}
			default:
				Err = ArgsErrMsg(append([]any{e}, args...)...)
			}
		}
	}

	exception := &ErrType{
		Err:         Err,
		MsgStack:    MsgStack,
		CallerStack: CallerStack,
		Position:    Position,
	}

	isErrGroup := false
	errHash := MD5(Err)
	if len(CallerStack) > 0 {
		errHash = MD5(CallerStack[0], Err)
		isErrGroup = strings.Contains(Err, bars) || strings.Contains(CallerStack[0], bars)
	}

	hub := sentry.CurrentHub()
	client, scope := hub.Client(), hub.Scope()
	// sentry.CaptureException(exception)
	// Warn("%s\n%#v\n|%s", Err, isErrGroup, CallerStack[0])

	sentryMux.Lock()
	if client != nil && scope != nil && sentryErrorMap[errHash] == 0 && !isErrGroup {
		event := client.EventFromException(exception, sentry.LevelError)

		e := event.Exception[0]
		l := len(e.Stacktrace.Frames)

		e.Stacktrace.Frames = lo.Filter(e.Stacktrace.Frames, func(e sentry.Frame, i int) bool {
			return i < l-2
		})

		e.Type = e.Stacktrace.Frames[len(e.Stacktrace.Frames)-1].Function
		event.Exception[0] = e

		sentryEvent := &SentryEvent{event, scope, exception, errHash}
		sentryEvents = append(sentryEvents, sentryEvent)
		sentryErrorMap[errHash] = time.Now().Unix()
	}
	sentryMux.Unlock()

	return exception
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
		if IsDebugLow() {
			ZLogErr.Err(err.DebugError()).Msg(msg)
		} else {
			ZLogErr.Err(err.OriginalError()).Msg(msg) // detailed error in STDERR
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

// ErrMsg returns a simple error message
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
		PrintFatal(e)
	}
	return assert.NoError(t, e)
}

// ErrorGroup represents a group of errors
type ErrorGroup struct {
	Logging bool
	Errors  []error
	Names   []string
}

// Add adds an error to the error group
func (e *ErrorGroup) Add(err ...error) {
	e.Errors = append(e.Errors, err...)
}

// Len returns the number of errors captured
func (e *ErrorGroup) Len() int {
	return len(e.Errors)
}

// Capture adds an error to the error group, and return true if err was not nil
func (e *ErrorGroup) Capture(err error, name ...string) bool {
	if err != nil {
		Name := ""
		if len(name) > 0 {
			Name = name[0]
		}
		e.Errors = append(e.Errors, err)
		e.Names = append(e.Names, Name)
		return true
	}
	return false
}

// Reset reset the errors to none
func (e *ErrorGroup) Reset() {
	e.Errors = []error{}
}

var bars = "---------------------------"

func (e *ErrorGroup) Error() string {
	return e.String()
}

func (e *ErrorGroup) MD5() string {
	parts := []string{}
	for _, err := range e.Errors {
		switch et := err.(type) {
		case ErrorInterface:
			parts = append(parts, et.MD5())
		default:
			parts = append(parts, MD5(err.Error()))
		}
	}
	return MD5(parts...)
}

func (e *ErrorGroup) Debug() string {
	if len(e.Errors) == 0 {
		return ""
	}

	errStringsMap := map[string]struct{}{}
	errStrings := []string{}
	for i, err := range e.Errors {
		if err == nil {
			continue
		}

		errString := "\n"
		if len(e.Names) == len(e.Errors) && e.Names[i] != "" {
			errString = F("\n%s %s %s\n", bars, e.Names[i], bars)
		}

		errHash := MD5(err.Error())
		switch et := err.(type) {
		case ErrorInterface:
			errHash = et.MD5()
			if IsDebug() {
				errString = errString + et.Debug()
			}
		default:
			errString = errString + err.Error()
		}

		if _, ok := errStringsMap[errHash]; !ok {
			errStrings = append(errStrings, errString)
		}
		errStringsMap[errHash] = struct{}{}
	}

	return strings.Join(errStrings, "\n")
}

// Err returns an error if any errors had been added
func (e *ErrorGroup) Err() error {
	if len(e.Errors) == 0 {
		return nil
	}
	// return e
	return fmt.Errorf(e.String())
}

func (e *ErrorGroup) String() string {
	if len(e.Errors) == 0 {
		return ""
	}

	errStringsMap := map[string]struct{}{}
	errStrings := []string{}
	for i, err := range e.Errors {
		if err == nil {
			continue
		}

		errString := "\n"
		if len(e.Names) == len(e.Errors) && e.Names[i] != "" {
			errString = F("\n%s %s %s\n", bars, e.Names[i], bars)
		}

		if err2, ok := err.(*ErrType); ok && IsDebug() {
			errString = errString + err2.Debug()
		} else {
			errString = errString + err.Error()
		}

		if _, ok := errStringsMap[errString]; !ok {
			errStrings = append(errStrings, errString)
		}
		errStringsMap[errString] = struct{}{}
	}

	return strings.Join(errStrings, "\n")
}

// ErrJSON returns to the echo.Context as JSON formatted
func ErrJSON(HTTPStatus int, err error, args ...interface{}) error {
	msg := ArgsErrMsg(args...)
	LogError(err)
	if msg == "" {
		msg = ErrMsg(err)
	} else if ErrMsg(err) != "" {
		msg = F("%s [%s]", msg, ErrMsg(err))
	}
	return NewHTTPError(HTTPStatus, M("error", msg))
}

// /////////////////////////// From Echo LabStack /////////////////////////////

type HTTPError struct {
	Code     int         `json:"-"`
	Message  interface{} `json:"message"`
	Internal error       `json:"-"` // Stores the error returned by an external dependency
}

// Error makes it compatible with `error` interface.
func (he *HTTPError) Error() string {
	if he.Internal == nil {
		return fmt.Sprintf("code=%d, message=%v", he.Code, he.Message)
	}
	return fmt.Sprintf("code=%d, message=%v, internal=%v", he.Code, he.Message, he.Internal)
}

// SetInternal sets error to HTTPError.Internal
func (he *HTTPError) SetInternal(err error) *HTTPError {
	he.Internal = err
	return he
}

// Unwrap satisfies the Go 1.13 error wrapper interface.
func (he *HTTPError) Unwrap() error {
	return he.Internal
}

// NewHTTPError creates a new HTTPError instance.
func NewHTTPError(code int, message ...interface{}) *HTTPError {
	he := &HTTPError{Code: code, Message: http.StatusText(code)}
	if len(message) > 0 {
		he.Message = message[0]
	}
	return he
}

var SentryRelease = ""
var SentryDsn = ""
var SentryConfigureFunc = func(event *SentryEvent) bool { return false }
var sentryErrorMap = map[string]int64{}
var sentryMux = sync.Mutex{}

type SentryEvent struct {
	Event     *sentry.Event
	Scope     *sentry.Scope
	Exception *ErrType
	Hash      string
}

var sentryEvents = []*SentryEvent{}

func SentryInit() {
	sentryOptions := sentry.ClientOptions{
		// or SENTRY_DSN environment variable
		Dsn: SentryDsn,
		// Either set environment and release here or set the SENTRY_ENVIRONMENT
		// and SENTRY_RELEASE environment variables.
		Environment: lo.Ternary(strings.HasSuffix(SentryRelease, "dev"), "Development", "Production"),
		Release:     SentryRelease,
		Debug:       false,
	}
	sentry.Init(sentryOptions)
}

func SentryFlush(timeout time.Duration) {
	hub := sentry.CurrentHub()
	client := hub.Client()
	captured := 0
	firstErr := ""
	for _, sentryEvent := range sentryEvents {
		if e := sentryEvent.Exception; captured > 0 {
			stack := Marshal(sentryEvent.Exception.Stack()) + " " + sentryEvent.Exception.Err
			if e.Err == "" || strings.Contains(stack, "context canceled") || (firstErr != "" && strings.Contains(stack, firstErr)) {
				continue // don't capture since we already have first
			}
		}

		if SentryConfigureFunc(sentryEvent) {
			hint := &sentry.EventHint{OriginalException: sentryEvent.Exception}
			client.CaptureEvent(sentryEvent.Event, hint, sentryEvent.Scope)
			captured++
			if firstErr == "" {
				firstErr = sentryEvent.Exception.Err
			}

			// Warn(">>>>>>" + Pretty(sentryEvent.Exception.Stack()))
			// Info(sentryEvent.Exception.Err)
		}
	}
	sentry.Flush(timeout)
}
