package process

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	g "github.com/flarco/g"
	"github.com/spf13/cast"
)

// Session is a session to execute processes and keep output
type Session struct {
	Proc           *Proc
	Alias          map[string]string
	Env            map[string]string
	Workdir        string
	Print          bool
	Stderr, Stdout string
	scanner        *ScanConfig
	mux            sync.Mutex
}

// Proc is a command background process
type Proc struct {
	Bin                        string
	Args                       []string
	Env                        map[string]string
	Err                        error
	Cmd                        *exec.Cmd
	WorkDir                    string
	HideCmdInErr               bool
	Print                      bool
	Stderr, Stdout, Combined   bytes.Buffer
	StderrReader, StdoutReader io.Reader
	StdinWriter                io.Writer
	StdinOverride              io.Reader
	Pid                        int
	Nice                       int
	Context                    *g.Context
	Done                       chan struct{} // finished with scanner
	exited                     chan struct{} // process exited
	scanner                    *ScanConfig
	printMux                   sync.Mutex
}

type ScanConfig struct {
	scanFunc func(stderr bool, text string)
}

// NewSession creates a session to execute processes
func NewSession() (s *Session) {
	s = &Session{
		Alias: map[string]string{},
		Env:   map[string]string{},
	}
	return
}

// AddAlias add an alias for a binary command str
func (s *Session) AddAlias(key, value string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.Alias[key] = value
}

// SetScanner sets scanner with the provided function
func (s *Session) SetScanner(scanFunc func(stderr bool, text string)) {
	s.scanner = &ScanConfig{scanFunc: scanFunc}
}

// Run runs a command
func (s *Session) Run(bin string, args ...string) (err error) {
	_, _, err = s.RunOutput(bin, args...)
	if err != nil {
		err = g.Error(err)
	}
	return
}

// RunOutput runs a command and returns the output
func (s *Session) RunOutput(bin string, args ...string) (stdout, stderr string, err error) {

	if val, ok := s.Alias[bin]; ok {
		bin = val
	}

	p, err := NewProc(bin, args...)
	if err != nil {
		err = g.Error(err, "could not init process")
		return
	}
	p.Env = s.Env
	p.WorkDir = s.Workdir
	p.scanner = s.scanner

	err = p.Run()
	if err != nil {
		e, ok := err.(*g.ErrType)
		if ok {
			// replace alias value with alias key in messages
			// this is to hide unneeded/unwanted details
			for i, msg := range e.MsgStack {
				for k, v := range s.Alias {
					msg = strings.ReplaceAll(msg, v, k)
				}
				e.MsgStack[i] = msg
			}
			err = g.Error(e, "error running process")
		} else {
			err = g.Error(err, "error running process")
		}
		return
	}

	sepStr := g.F(
		"%s %s",
		strings.Repeat("#", 5),
		p.String(),
	)

	// replace alias value with alias key in messages
	// this is to hide unneeded/unwanted details
	for k, v := range s.Alias {
		sepStr = strings.ReplaceAll(sepStr, v, k)
	}

	if s.Stdout == "" {
		s.Stdout = sepStr
	} else {
		s.Stdout = s.Stdout + "\n" + sepStr
	}

	if s.Stderr == "" {
		s.Stderr = sepStr
	} else {
		s.Stderr = s.Stderr + "\n" + sepStr
	}

	s.Stdout = s.Stdout + "\n" + p.Stdout.String()
	s.Stderr = s.Stderr + "\n" + p.Stderr.String()

	return
}

// NewProc creates a new process and returns it
func NewProc(bin string, args ...string) (p *Proc, err error) {
	p = &Proc{
		Bin:    bin,
		Args:   args,
		Done:   make(chan struct{}),
		exited: make(chan struct{}),
		Env:    map[string]string{},
	}

	if !p.ExecutableFound() {
		err = g.Error("executable '%s' not found in path", p.Bin)
	}
	return
}

// String returns the command as a string
func (p *Proc) String() string {
	parts := []string{p.Bin}
	for _, a := range p.Args {
		if strings.Contains(a, `"`) {
			a = `"` + strings.ReplaceAll(a, `"`, `""`) + `"`
		}
		parts = append(parts, a)
	}
	return strings.Join(parts, " ")
}

// ExecutableFound returns true if the executable is found
func (p *Proc) ExecutableFound() bool {
	_, err := exec.LookPath(p.Bin)
	return err == nil
}

// SetArgs sets the args for the command
func (p *Proc) SetArgs(args ...string) {
	p.Args = args
}

func (p *Proc) Close() (err error) {
	wc, ok := p.StdinWriter.(io.WriteCloser)
	if ok {
		err = wc.Close()
		if err != nil {
			return g.Error(err, "could not close StdinPipe")
		}
	} else {
		g.Debug("could not cast to io.WriteCloser")
	}
	return nil
}

// Start executes the command
func (p *Proc) Start(args ...string) (err error) {
	if len(args) > 0 {
		p.SetArgs(args...)
	}

	if p.Context == nil {
		context := g.NewContext(context.Background())
		p.Context = &context
	}

	// reset channels
	p.Done = make(chan struct{})
	p.exited = make(chan struct{})

	p.Cmd = exec.Command(p.Bin, p.Args...)
	p.Cmd.Dir = p.WorkDir
	if p.Env != nil {
		p.Cmd.Env = g.MapToKVArr(p.Env)
	}

	p.Stdout.Reset()
	p.Stderr.Reset()
	p.Combined.Reset()

	p.StdoutReader, err = p.Cmd.StdoutPipe()
	if err != nil {
		return g.Error(err)
	}
	p.StderrReader, err = p.Cmd.StderrPipe()
	if err != nil {
		return g.Error(err)
	}

	if p.StdinOverride != nil {
		p.Cmd.Stdin = p.StdinOverride
	} else {
		p.StdinWriter, err = p.Cmd.StdinPipe()
		if err != nil {
			return g.Error(err)
		}
	}

	g.Trace("Proc command -> %s", p.String())
	err = p.Cmd.Start()
	if err != nil {
		return g.Error(err, p.CmdErrorText())
	}

	p.Pid = p.Cmd.Process.Pid

	go p.scanAndWait()

	// set NICE
	if runtime.GOOS != "windows" && p.Nice != 0 {
		niceCmd := exec.Command("renice", "-n", cast.ToString(p.Nice), "-p", cast.ToString(p.Pid))
		niceCmd.Run()
	}

	// listen for context cancel
	go func() {
		select {
		case <-p.Done:
			return
		case <-p.Context.Ctx.Done():
		}

		g.Debug("interrupting sub-process %d", p.Cmd.Process.Pid)
		p.Cmd.Process.Signal(syscall.SIGINT)
		t := time.NewTimer(5 * time.Second)
		select {
		case <-p.Done:
			return
		case <-t.C:
			g.Debug("killing sub-process %d", p.Cmd.Process.Pid)
			g.LogError(p.Cmd.Process.Kill())
		}
	}()

	return
}

// SetScanner sets scanner with the provided function
func (p *Proc) SetScanner(scanFunc func(stderr bool, text string)) {
	p.scanner = &ScanConfig{scanFunc: scanFunc}
}

func (p *Proc) scanAndWait() {

	readLine := func(r *bufio.Reader, stderr bool) error {
		text, err := r.ReadString('\n')
		if err != nil {
			return err
		}
		text = strings.TrimSuffix(text, "\n")

		p.printMux.Lock()
		if p.Print {
			if stderr {
				fmt.Fprintf(os.Stderr, "%s\n", text)
			} else {
				fmt.Fprintf(os.Stdout, "%s\n", text)
			}
		}
		if p.scanner != nil {
			p.scanner.scanFunc(stderr, text)
		}
		p.printMux.Unlock()

		return nil
	}

	scanLoop := func(exited bool) {
		stderrTo := make(chan bool)
		stdoutTo := make(chan bool)

		var t *time.Timer
		if exited {
			t = time.NewTimer(time.Second)
		} else {
			t = time.NewTimer(999999 * time.Hour) // infinite if not exited
		}
		stdoutReader := bufio.NewReader(p.StdoutReader)
		stderrReader := bufio.NewReader(p.StderrReader)

		readStdErr := func() {
			var err error
			for err == nil {
				err = readLine(stderrReader, true)
			}
			stderrTo <- true
		}

		readStdOut := func() {
			var err error
			for err == nil {
				err = readLine(stdoutReader, false)
			}
			stdoutTo <- true
		}

		go readStdErr()
		go readStdOut()

		for {
			select {
			case <-t.C:
				return
			case <-p.exited:
				return
			case <-stderrTo:
				go readStdErr()
			case <-stdoutTo:
				go readStdOut()
			}
		}
	}

	go func() {
		err := p.Cmd.Wait()
		if err != nil {
			p.Err = g.Error(err, p.CmdErrorText())
		}
		close(p.exited)
	}()

	scanLoop(false)

	<-p.exited

	scanLoop(true) // drain remaining

	close(p.Done)
}

// Run executes a command, prints output and waits for it to finish
func (p *Proc) Run(args ...string) (err error) {
	err = p.Start(args...)
	if err != nil {
		err = g.Error(err, "could not start process. %s", p.CmdErrorText())
		return
	}

	err = p.Wait()
	if err != nil {
		return g.Error(err)
	}

	return
}

// CmdErrorText returns the command error text
func (p *Proc) CmdErrorText() string {
	if p.HideCmdInErr {
		return g.F("%s\n%s", p.Stderr.String(), p.Stdout.String())
	}
	return fmt.Sprintf(
		"Proc command -> %s\n%s\n%s",
		p.String(), p.Stderr.String(), p.Stdout.String(),
	)
}

// StdoutScannerLines returns a scanner for stdout
func (p *Proc) StdoutScannerLines() (scanner *bufio.Scanner) {
	if p.StdoutReader == nil {
		return
	}
	scanner = bufio.NewScanner(p.StdoutReader)
	scanner.Split(bufio.ScanLines)
	return scanner
}

// StderrScannerLines returns a scanner for stderr
func (p *Proc) StderrScannerLines() (scanner *bufio.Scanner) {
	if p.StderrReader == nil {
		return
	}
	scanner = bufio.NewScanner(p.StderrReader)
	scanner.Split(bufio.ScanLines)
	return scanner
}

// Wait waits for the process to end
func (p *Proc) Wait() error {

	<-p.Done
	code := p.Cmd.ProcessState.ExitCode()
	if p.Err != nil {
		return g.Error(p.Err)
	} else if code != 0 {
		return g.Error("exit code = %d.\n%s", code, p.CmdErrorText())
	}

	return nil
}
