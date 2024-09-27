package process

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	g "github.com/flarco/g"
	"github.com/stretchr/testify/assert"
)

func TestProcess(t *testing.T) {
	t.Run("NewProc", func(t *testing.T) {
		p, err := NewProc("echo", "Hello, World!")
		assert.NoError(t, err)
		assert.NotNil(t, p)
	})

	t.Run("Proc.Run", func(t *testing.T) {
		p, _ := NewProc("echo", "Hello, World!")
		p.Capture = true
		err := p.Run()
		assert.NoError(t, err)
		assert.Contains(t, p.Stdout.String(), "Hello, World!")
	})

	t.Run("NonExistentCommand", func(t *testing.T) {
		p, err := NewProc("non_existent_command")
		assert.NotNil(t, p)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "executable 'non_existent_command' not found in path")
	})

	t.Run("Proc.SetScanner", func(t *testing.T) {
		p, _ := NewProc("echo", "Hello, Scanner!")
		scanCount := 0

		p.SetScanner(func(stderr bool, text string) {
			scanCount++
			assert.False(t, stderr)
			assert.Contains(t, text, "Hello, Scanner!")
		})
		p.Capture = true

		err := p.Run()
		assert.NoError(t, err)
		assert.Equal(t, 1, scanCount)
		assert.Contains(t, p.Stdout.String(), "Hello, Scanner!")
	})

	t.Run("Proc.SetScannerMultipleLines", func(t *testing.T) {
		p, _ := NewProc("echo", "-e", "Line 1\nLine 2\nLine 3")
		lineCount := 0

		p.SetScanner(func(stderr bool, text string) {
			lineCount++
			assert.False(t, stderr)
			assert.Contains(t, text, "Line")
		})
		p.Capture = true

		err := p.Run()
		assert.NoError(t, err)
		assert.Equal(t, 3, lineCount)
		assert.Contains(t, p.Stdout.String(), "Line 1")
		assert.Contains(t, p.Stdout.String(), "Line 2")
		assert.Contains(t, p.Stdout.String(), "Line 3")
	})

	t.Run("ProcWithStdinInput", func(t *testing.T) {
		p, _ := NewProc("cat")
		p.Capture = true
		p.StdinOverride = strings.NewReader("Hello from stdin!")

		err := p.Run()
		assert.NoError(t, err)
		assert.Contains(t, p.Stdout.String(), "Hello from stdin!")
	})

	t.Run("ProcWithMultiLineStdinInput", func(t *testing.T) {
		p, _ := NewProc("sort")
		p.Capture = true
		p.StdinOverride = strings.NewReader("banana\napple\ncherry")

		err := p.Run()
		assert.NoError(t, err)
		assert.Equal(t, "apple\nbanana\ncherry\n", p.Stdout.String())
	})

	t.Run("ProcWithStdinAndArguments", func(t *testing.T) {
		p, _ := NewProc("grep", "Hello")
		p.Capture = true
		p.StdinOverride = strings.NewReader("Hello, World!\nGoodbye, World!")

		err := p.Run()
		assert.NoError(t, err)
		assert.Equal(t, "Hello, World!\n", p.Stdout.String())
	})

	t.Run("ProcWithStdinWriter", func(t *testing.T) {
		p, _ := NewProc("cat")
		p.Capture = true

		err := p.Start()
		assert.NoError(t, err)

		_, err = p.StdinWriter.Write([]byte("Hello from StdinWriter!"))
		assert.NoError(t, err)

		err = p.CloseStdin()
		assert.NoError(t, err)

		err = p.Wait()
		assert.NoError(t, err)
		assert.Contains(t, p.Stdout.String(), "Hello from StdinWriter!")
	})

	t.Run("ProcWithMultiLineStdinWriter", func(t *testing.T) {
		p, _ := NewProc("sort")
		p.Capture = true

		err := p.Start()
		assert.NoError(t, err)

		_, err = p.StdinWriter.Write([]byte("banana\napple\ncherry"))
		assert.NoError(t, err)

		err = p.CloseStdin()
		assert.NoError(t, err)

		err = p.Wait()
		assert.NoError(t, err)
		assert.Equal(t, "apple\nbanana\ncherry\n", p.Stdout.String())
	})

	t.Run("ProcWithScannerAndStdinWriter", func(t *testing.T) {
		p, err := NewProc("cat")
		assert.NoError(t, err)
		p.Capture = true

		scannerOutput := ""
		p.SetScanner(func(isStderr bool, text string) {
			if !isStderr {
				scannerOutput += text + "\n"
			}
		})

		err = p.Start()
		assert.NoError(t, err)

		inputText := "Hello from StdinWriter!\nThis is a test.\n"
		_, err = p.StdinWriter.Write([]byte(inputText))
		assert.NoError(t, err)

		// Give some time for the scanner to process the input
		time.Sleep(100 * time.Millisecond)

		err = p.CloseStdin()
		assert.NoError(t, err)

		err = p.Wait()
		assert.NoError(t, err)

		assert.Equal(t, inputText, p.Stdout.String())
		assert.Equal(t, inputText, scannerOutput)
	})
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
	p.Context = g.NewContext(c)

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
