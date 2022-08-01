package g

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStats(t *testing.T) {
	pStats := GetProcStats(os.Getpid())
	assert.NotEmpty(t, pStats)
	PP(pStats)

	stats := GetMachineProcStats()
	assert.NotEmpty(t, stats)
	PP(stats)
}
