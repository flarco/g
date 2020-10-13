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
	proc.SetScanner(func(se bool, t string) { c++ })
	err = proc.Start()
	assert.NoError(t, err)

	err = proc.Wait()
	assert.NoError(t, err)

	assert.Greater(t, c, 0)
}

func TestSession(t *testing.T) {
	sess := NewSession()

	err := sess.Run("ls", "-l")
	assert.NoError(t, err)

	c := 0
	sess.SetScanner(func(se bool, t string) { c++ })
	err = sess.Run("ls", "-l", "/")
	assert.NoError(t, err)

	assert.Greater(t, c, 0)

	// println(sess.Stdout)
}
