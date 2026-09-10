package g

import (
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoHooksStripsMapArgs(t *testing.T) {
	prev := LogHooks
	t.Cleanup(func() { LogHooks = prev })

	var got *LogLine
	LogHooks = []*LogHook{{
		Level: zerolog.InfoLevel,
		Func: func(ll *LogLine) {
			copy := *ll
			got = &copy
		},
	}}

	Info("hello %s", "world", map[string]interface{}{"job_id": "j1"})
	require.NotNil(t, got)
	line := got.Line()
	assert.NotContains(t, line, "%!(EXTRA")
	assert.True(t, strings.Contains(line, "hello world"), line)
}

func TestDoHooksValues(t *testing.T) {
	prev := LogHooks
	t.Cleanup(func() { LogHooks = prev })

	var got *LogLine
	LogHooks = []*LogHook{{
		Level: zerolog.InfoLevel,
		Func: func(ll *LogLine) {
			copy := *ll
			if ll.Values != nil {
				vals := map[string]any{}
				for k, v := range ll.Values {
					vals[k] = v
				}
				copy.Values = vals
			}
			got = &copy
		},
	}}

	Info("hello %s", "world", map[string]interface{}{"job_id": "j1", "project_id": "p1"})
	require.NotNil(t, got)
	require.NotNil(t, got.Values)
	assert.Equal(t, "j1", got.Values["job_id"])
	assert.Equal(t, "p1", got.Values["project_id"])
	assert.Equal(t, []any{"world"}, got.Args)
	assert.NotContains(t, got.Line(), "%!(EXTRA")
	assert.Contains(t, got.Line(), "hello world")
}

func TestContextInfoHookNoExtraMap(t *testing.T) {
	prev := LogHooks
	t.Cleanup(func() { LogHooks = prev })

	var got *LogLine
	LogHooks = []*LogHook{{
		Level: zerolog.InfoLevel,
		Func: func(ll *LogLine) {
			copy := *ll
			got = &copy
		},
	}}

	ctx := NewContext(context.Background())
	ctx.SetLogValues("job_id", "j1")
	ctx.Info("started")
	require.NotNil(t, got)
	assert.NotContains(t, got.Line(), "%!(EXTRA")
}
