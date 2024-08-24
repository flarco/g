package process

import (
	"context"
	"runtime"
	"testing"
	"time"

	g "github.com/flarco/g"
	"github.com/stretchr/testify/assert"
)

func TestProcess(t *testing.T) {
	// Test NewProc
	p, err := NewProc("echo", "Hello, World!")
	assert.NoError(t, err)
	assert.NotNil(t, p)

	// Test Proc.Run
	p.Capture = true
	err = p.Run()
	assert.NoError(t, err)
	assert.Contains(t, p.Stdout.String(), "Hello, World!")

	// Test Proc with non-existent command
	p, err = NewProc("non_existent_command")
	assert.NotNil(t, p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executable 'non_existent_command' not found in path")

	// Test Proc.SetScanner
	p, err = NewProc("echo", "Hello, Scanner!")
	assert.NoError(t, err)
	assert.NotNil(t, p)

	scanCount := 0
	p.SetScanner(func(stderr bool, text string) {
		scanCount++
		assert.False(t, stderr)
		assert.Contains(t, text, "Hello, Scanner!")
	})

	p.Capture = true
	err = p.Run()
	assert.NoError(t, err)
	assert.Equal(t, 1, scanCount)
	assert.Contains(t, p.Stdout.String(), "Hello, Scanner!")

	// Test Proc.SetScanner with multiple lines
	p, err = NewProc("echo", "-e", "Line 1\nLine 2\nLine 3")
	assert.NoError(t, err)
	assert.NotNil(t, p)

	lineCount := 0
	p.SetScanner(func(stderr bool, text string) {
		lineCount++
		assert.False(t, stderr)
		assert.Contains(t, text, "Line")
	})

	p.Capture = true
	err = p.Run()
	assert.NoError(t, err)
	assert.Equal(t, 3, lineCount)
	assert.Contains(t, p.Stdout.String(), "Line 1")
	assert.Contains(t, p.Stdout.String(), "Line 2")
	assert.Contains(t, p.Stdout.String(), "Line 3")
}

func TestSession(t *testing.T) {
	sess := NewSession()
	sess.Capture = true

	// Test basic Run
	err := sess.Run("echo", "Hello")
	assert.NoError(t, err)
	assert.Contains(t, sess.Stdout, "Hello")

	// Test SetScanner
	c := 0
	sess.SetScanner(func(se bool, t string) { c++ })
	err = sess.Run("ls", "-l", "/")
	assert.NoError(t, err)
	assert.Greater(t, c, 0)

	// Test AddAlias
	sess.AddAlias("greet", "echo")
	err = sess.Run("greet", "Hello")
	assert.NoError(t, err)
	assert.Contains(t, sess.Stdout, "Hello")

	// Test RunOutput
	stdout, stderr, err := sess.RunOutput("echo", "Test output")
	assert.NoError(t, err)
	assert.Contains(t, stdout, "Test output")
	assert.Empty(t, stderr)
}

func TestProcWithContext(t *testing.T) {
	p, err := NewProc("sleep", "5")
	assert.NoError(t, err)

	c, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	ctx := g.NewContext(c)
	p.Context = &ctx

	start := time.Now()
	err = p.Run()
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signal: interrupt")
	assert.Less(t, duration, 2*time.Second)
}

func TestProcWithNice(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	p, err := NewProc("sleep", "1")
	assert.NoError(t, err)

	p.Nice = 10
	err = p.Run()
	assert.NoError(t, err)

	// We can't easily verify the nice value, but we can check that the command ran successfully
}

func TestGetParent(t *testing.T) {
	parent := GetParent()
	assert.NotZero(t, parent.PID)
	assert.NotEmpty(t, parent.Name)
	assert.NotEmpty(t, parent.Executable)
}
