package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcess(t *testing.T) {
	proc, err := NewProc("ls", "-l")
	assert.NoError(t, err)

	err = proc.Run()
	assert.NoError(t, err)

	c := 0
	proc.SetScanner(true, true, func(s, t string) { c++ })
	err = proc.Start()
	assert.NoError(t, err)

	err = proc.Wait()
	assert.NoError(t, err)

	assert.Greater(t, c, 0)
}
