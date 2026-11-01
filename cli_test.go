package g

import (
	"testing"

	"github.com/integrii/flaggy"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCliProcessValues(t *testing.T) {
	for _, withoutFlags := range []bool{true, false} {
		AllScs = nil
		var got map[string]any
		sc := (&CliSC{
			Name:                "patch",
			ExecuteWithoutFlags: withoutFlags,
			Flags: []Flag{
				{Name: "set", Type: "slice"},
				{Name: "file"},
				{Name: "force", Type: "bool"},
			},
			ExecProcess: func(c *CliSC) (bool, error) {
				got = c.Vals
				return true, nil
			},
		}).Make()
		parser := flaggy.NewParser("test")
		parser.AttachSubcommand(sc.Sc, 1)
		require.NoError(t, parser.ParseArgs([]string{"patch", "--set", "a.b=true", "--set", "c=1", "--force"}))

		ok, err := CliProcess()
		require.NoError(t, err)
		require.True(t, ok, "without flags: %t", withoutFlags)
		assert.Equal(t, []string{"a.b=true", "c=1"}, got["set"], "without flags: %t", withoutFlags)
		assert.True(t, cast.ToBool(got["force"]))
		assert.NotContains(t, got, "file", "a blank flag is removed")
	}
	AllScs = nil
}
