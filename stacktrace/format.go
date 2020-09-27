package stacktrace

import (
	"fmt"
	"strings"
)

/*
DefaultFormat defines the behavior of err.Error() when called on a stacktrace,
as well as the default behavior of the "%v", "%s" and "%q" formatting
specifiers. By default, all of these produce a full stacktrace including line
number information. To have them produce a condensed single-line output, set
this value to stacktrace.FormatBrief.

The formatting specifier "%+s" can be used to force a full stacktrace regardless
of the value of DefaultFormat. Similarly, the formatting specifier "%#s" can be
used to force a brief output.
*/
var DefaultFormat = FormatFull

// Format is the type of the two possible values of stacktrace.DefaultFormat.
type Format int

const (
	// FormatFull means format as a full stacktrace including line number information.
	FormatFull Format = iota
	// FormatBrief means Format on a single line without line number information.
	FormatBrief
)

var _ fmt.Formatter = (*stacktrace)(nil)

func (st *stacktrace) Format(f fmt.State, c rune) {
	var text string
	if f.Flag('+') && !f.Flag('#') && c == 's' { // "%+s"
		text = formatFull(st)
	} else if f.Flag('#') && !f.Flag('+') && c == 's' { // "%#s"
		text = formatBrief(st)
	} else {
		text = map[Format]func(*stacktrace) string{
			FormatFull:  formatFull,
			FormatBrief: formatBrief,
		}[DefaultFormat](st)
	}

	formatString := "%"
	// keep the flags recognized by fmt package
	for _, flag := range "-+# 0" {
		if f.Flag(int(flag)) {
			formatString += string(flag)
		}
	}
	if width, has := f.Width(); has {
		formatString += fmt.Sprint(width)
	}
	if precision, has := f.Precision(); has {
		formatString += "."
		formatString += fmt.Sprint(precision)
	}
	formatString += string(c)
	fmt.Fprintf(f, formatString, text)
}

func formatFull(st *stacktrace) string {
	var str string
	newline := func() {
		if str != "" && !strings.HasSuffix(str, "\n") {
			str += "\n"
		}
	}

	for curr, ok := st, true; ok; curr, ok = curr.cause.(*stacktrace) {
		str += curr.message

		if curr.file != "" {
			newline()
			if curr.function == "" {
				str += fmt.Sprintf(" --- at %v:%v ---", curr.file, curr.line)
			} else {
				str += fmt.Sprintf(" --- at %v:%v (%v) ---", curr.file, curr.line, curr.function)
			}
		}

		if curr.cause != nil {
			newline()
			if cause, ok := curr.cause.(*stacktrace); !ok {
				str += "Caused by: "
				str += curr.cause.Error()
			} else if cause.message != "" {
				str += "Caused by: "
			}
		}
	}

	return str
}

func formatBrief(st *stacktrace) string {
	var str string
	concat := func(msg string) {
		if str != "" && msg != "" {
			str += ": "
		}
		str += msg
	}

	curr := st
	for {
		concat(curr.message)
		if cause, ok := curr.cause.(*stacktrace); ok {
			curr = cause
		} else {
			break
		}
	}
	if curr.cause != nil {
		concat(curr.cause.Error())
	}
	return str
}
