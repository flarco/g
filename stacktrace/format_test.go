package stacktrace_test

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/palantir/stacktrace"
)

func TestFormat(t *testing.T) {
	plainErr := errors.New("plain")
	stacktraceErr := stacktrace.Propagate(plainErr, "decorated")
	digits := regexp.MustCompile(`\d`)

	for _, test := range []struct {
		format             stacktrace.Format
		specifier          string
		expectedPlain      string
		expectedStacktrace string
	}{
		{
			format:             stacktrace.FormatFull,
			specifier:          "%v",
			expectedPlain:      "plain",
			expectedStacktrace: "decorated\n --- at github.com/palantir/stacktrace/format_test.go:## (TestFormat) ---\nCaused by: plain",
		},
		{
			format:             stacktrace.FormatFull,
			specifier:          "%q",
			expectedPlain:      "\"plain\"",
			expectedStacktrace: "\"decorated\\n --- at github.com/palantir/stacktrace/format_test.go:## (TestFormat) ---\\nCaused by: plain\"",
		},
		{
			format:             stacktrace.FormatFull,
			specifier:          "%105s",
			expectedPlain:      "                                                                                                    plain",
			expectedStacktrace: "     decorated\n --- at github.com/palantir/stacktrace/format_test.go:## (TestFormat) ---\nCaused by: plain",
		},
		{
			format:             stacktrace.FormatFull,
			specifier:          "%#s",
			expectedPlain:      "plain",
			expectedStacktrace: "decorated: plain",
		},
		{
			format:             stacktrace.FormatBrief,
			specifier:          "%v",
			expectedPlain:      "plain",
			expectedStacktrace: "decorated: plain",
		},
		{
			format:             stacktrace.FormatBrief,
			specifier:          "%q",
			expectedPlain:      "\"plain\"",
			expectedStacktrace: "\"decorated: plain\"",
		},
		{
			format:             stacktrace.FormatBrief,
			specifier:          "%20s",
			expectedPlain:      "               plain",
			expectedStacktrace: "    decorated: plain",
		},
		{
			format:             stacktrace.FormatBrief,
			specifier:          "%+s",
			expectedPlain:      "plain",
			expectedStacktrace: "decorated\n --- at github.com/palantir/stacktrace/format_test.go:## (TestFormat) ---\nCaused by: plain",
		},
	} {
		stacktrace.DefaultFormat = test.format

		actualPlain := fmt.Sprintf(test.specifier, plainErr)
		assert.Equal(t, test.expectedPlain, actualPlain)

		actualStacktrace := fmt.Sprintf(test.specifier, stacktraceErr)
		actualStacktrace = digits.ReplaceAllString(actualStacktrace, "#")
		assert.Equal(t, test.expectedStacktrace, actualStacktrace)
	}
}
