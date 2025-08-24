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
	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cast"
)

// Session is a session to execute processes and keep output
type Session struct {
	Proc           *Proc
	Alias          map[string]string
	Env            map[string]string
	Workdir        string
	Capture, Print bool
	Stderr, Stdout string
	scanner        *ScanConfig
	mux            sync.Mutex
}

type Label struct {
	Value string
	Len   int
	Color int
}

func (l *Label) Render() string {
	if l.Value == "" {
		return ""
	}

	label := l.Value
	if l.Color > 0 {
		label = g.Colorize(l.Color, l.Value)
	}

	if l.Len > len(l.Value) {
		label = label + strings.Repeat(" ", l.Len-len(l.Value))
	}
	return label
}

// Proc is a command background process
type Proc struct {
	Bin                          string
	Args                         []string
	Env                          map[string]string
	Err                          error
	Cmd                          *exec.Cmd
	Label                        Label
	WorkDir                      string
	HideCmdInErr                 bool
	Capture, Print               bool
	Stderr, Stdout, Combined     bytes.Buffer
	StdinOverride                io.Reader
	StderrReader, StdoutReader   io.ReadCloser
	stderrScanner, stdoutScanner *bufio.Scanner
	StdinWriter                  io.Writer
	Pid                          int
	Nice                         int
	Context                      *g.Context
	Done                         chan struct{} // finished with scanner
	scanner                      *ScanConfig
	printMux                     sync.Mutex
	tempScriptFile               string // path to temp script file for cleanup
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
	p.Capture = s.Capture
	p.Print = s.Print

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

	stdout = p.Stdout.String()
	stderr = p.Stderr.String()

	s.Stdout = s.Stdout + "\n" + stdout
	s.Stderr = s.Stderr + "\n" + stderr

	return
}

// NewProc creates a new process and returns it
func NewProc(bin string, args ...string) (p *Proc, err error) {
	p = &Proc{
		Bin:  bin,
		Args: args,
		Done: make(chan struct{}),
		Env:  map[string]string{},
	}

	if !p.ExecutableFound() {
		err = g.Error("executable '%s' not found in path", p.Bin)
	}
	return
}

// NewScript creates a new process that runs a script with multiple commands
// The script will exit on first error (equivalent to 'set -e' in bash)
// The temporary script file is automatically cleaned up when CleanupScript() is called
func NewScript(script string) (p *Proc, err error) {
	var tmpFile *os.File
	var content string
	var bin string
	var args []string

	if runtime.GOOS == "windows" {
		// Use PowerShell on Windows with error handling
		tmpFile, err = os.CreateTemp("", "script_*.ps1")
		if err != nil {
			return nil, g.Error(err, "could not create temp PowerShell script file")
		}

		bin = "powershell"
		args = []string{"-ExecutionPolicy", "Bypass", "-File", tmpFile.Name()}
		content = fmt.Sprintf(`$ErrorActionPreference = "Stop"
%s`, script)
	} else {
		// Use bash on Linux/Mac with error handling
		tmpFile, err = os.CreateTemp("", "script_*.sh")
		if err != nil {
			return nil, g.Error(err, "could not create temp bash script file")
		}

		bin = "bash"
		args = []string{tmpFile.Name()}
		content = fmt.Sprintf(`#!/bin/bash
set -e
%s`, script)
	}

	// Write script content to temp file
	_, err = tmpFile.WriteString(content)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, g.Error(err, "could not write to temp script file")
	}
	tmpFile.Close()

	// Make executable on Unix systems
	if runtime.GOOS != "windows" {
		err = os.Chmod(tmpFile.Name(), 0755)
		if err != nil {
			os.Remove(tmpFile.Name())
			return nil, g.Error(err, "could not make script executable")
		}
	}

	// Create process
	p, err = NewProc(bin, args...)
	if err != nil {
		os.Remove(tmpFile.Name())
		return nil, g.Error(err, "could not create process for script")
	}

	// Store temp file path for cleanup
	p.tempScriptFile = tmpFile.Name()

	return p, nil
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

// CloseStdin closes the stdin pipe
func (p *Proc) CloseStdin() (err error) {
	wc, ok := p.StdinWriter.(io.WriteCloser)
	if ok {
		err = wc.Close()
		if err != nil {
			return g.Error(err, "could not close StdinPipe")
		}
	}
	return nil
}

// CleanupScript removes the temporary script file if it exists
func (p *Proc) CleanupScript() error {
	if p.tempScriptFile != "" {
		err := os.Remove(p.tempScriptFile)
		if err != nil && !os.IsNotExist(err) {
			return g.Error(err, "could not remove temp script file")
		}
		p.tempScriptFile = ""
	}
	return nil
}

// Start executes the command
func (p *Proc) Start(args ...string) (err error) {
	if len(args) > 0 {
		p.SetArgs(args...)
	}

	if p.Context == nil {
		p.Context = g.NewContext(context.Background())
	}

	// reset channels
	p.Done = make(chan struct{})

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

	p.stderrScanner = bufio.NewScanner(p.StderrReader)
	p.stderrScanner.Split(bufio.ScanLines)

	p.stdoutScanner = bufio.NewScanner(p.StdoutReader)
	p.stdoutScanner.Split(bufio.ScanLines)

	if p.StdinOverride != nil {
		p.Cmd.Stdin = p.StdinOverride
	} else {
		p.StdinWriter, err = p.Cmd.StdinPipe()
		if err != nil {
			return g.Error(err)
		}
	}

	g.Trace("Proc command -> %s", p.String())

	tries := 0
retry:
	tries++

	err = p.Cmd.Start()
	if err != nil {
		if strings.Contains(err.Error(), "text file busy") && tries < 10 {
			g.Warn("could not start command %s (%s), retrying...", p.String(), err.Error())
			time.Sleep(1 * time.Second)
			goto retry
		}
		return g.Error(err, p.CmdErrorText())
	}

	p.Pid = p.Cmd.Process.Pid

	go p.scanAndWait()

	// set NICE
	if runtime.GOOS != "windows" && p.Nice != 0 {
		niceCmd := exec.Command("renice", "-n", cast.ToString(p.Nice), "-p", cast.ToString(p.Pid))
		niceCmd.Run()
	}

	return
}

// SetScanner sets scanner with the provided function
func (p *Proc) SetScanner(scanFunc func(stderr bool, text string)) {
	p.printMux.Lock()
	p.scanner = &ScanConfig{scanFunc: scanFunc}
	p.printMux.Unlock()
}

// ResetBuffers clears the buffers
func (p *Proc) ResetBuffers() {
	p.Stdout.Reset()
	p.Stderr.Reset()
	p.Combined.Reset()
}

func (p *Proc) Exited() bool {
	return p.Cmd == nil || (p.Cmd.ProcessState != nil && p.Cmd.ProcessState.Exited())
}

func (p *Proc) scanAndWait() {

	scannerExitChan := make(chan bool)

	label := p.Label.Render()

	go func() {
		for p.stderrScanner.Scan() {
			line := p.stderrScanner.Text()
			p.printMux.Lock()
			if p.Capture {
				p.Stderr.WriteString(line + "\n")
				p.Combined.WriteString(line + "\n")
			}
			if p.scanner != nil && p.scanner.scanFunc != nil {
				p.scanner.scanFunc(true, line)
			}
			if p.Print {
				if label != "" {
					line = g.F("%s | %s", g.Colorize(g.ColorDarkGray, label), line)
				}
				fmt.Fprintf(os.Stderr, "%s", line+"\n")
			}
			p.printMux.Unlock()
		}
		scannerExitChan <- true
	}()

	go func() {
		for p.stdoutScanner.Scan() {
			line := p.stdoutScanner.Text()
			p.printMux.Lock()
			if p.Capture {
				p.Stdout.WriteString(line + "\n")
				p.Combined.WriteString(line + "\n")
			}
			if p.scanner != nil && p.scanner.scanFunc != nil {
				p.scanner.scanFunc(false, line)
			}
			if p.Print {
				if label != "" {
					line = g.F("%s | %s", g.Colorize(g.ColorDarkGray, label), line)
				}
				fmt.Fprintf(os.Stdout, "%s", line+"\n")
			}
			p.printMux.Unlock()
		}
		scannerExitChan <- true
	}()

	err := p.Cmd.Wait()
	if err != nil {
		p.Err = g.Error(err, p.CmdErrorText())
	}

	// wait for scanners to exit
	<-scannerExitChan
	<-scannerExitChan

	p.Done <- struct{}{}
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
		e := strings.TrimSpace(p.Stderr.String())
		o := strings.TrimSpace(p.Stdout.String())
		switch {
		case e == "":
			return o
		case o == "":
			return e
		}
		return e + "  " + o
	}
	return fmt.Sprintf(
		"Proc command -> %s\n%s\n%s",
		p.String(), p.Stderr.String(), p.Stdout.String(),
	)
}

// Wait waits for the process to end
func (p *Proc) Wait() error {

	select {
	case <-p.Done:
	case <-p.Context.Ctx.Done():
		g.Debug("interrupting sub-process %d", p.Cmd.Process.Pid)
		p.Cmd.Process.Signal(syscall.SIGINT)
		t := time.NewTimer(5 * time.Second)
		select {
		case <-p.Done:
		case <-t.C:
			g.Debug("killing sub-process %d", p.Cmd.Process.Pid)
			g.LogError(p.Cmd.Process.Kill())
		}
	}

	// Clean up temporary script file if it exists
	defer func() {
		if p.tempScriptFile != "" {
			g.LogError(p.CleanupScript())
		}
	}()

	if p.Err != nil {
		return p.Err
	}

	if p.Cmd != nil && p.Cmd.ProcessState != nil {
		if code := p.Cmd.ProcessState.ExitCode(); code != 0 {
			return g.Error("exit code = %d.\n%s", code, p.CmdErrorText())
		}
	}

	return nil
}

type Parent struct {
	PID        int      `json:"pid"`
	Name       string   `json:"name"`
	Executable string   `json:"executable"`
	Arguments  []string `json:"arguments"`
}

func GetParent() (parent Parent) {
	parent.PID = os.Getppid()

	p, err := process.NewProcess(cast.ToInt32(parent.PID))
	if err == nil {
		parent.Name, _ = p.Name()
		parent.Executable, _ = p.Exe()
		args, _ := p.CmdlineSlice()
		if len(args) > 1 && g.HasPrefix(strings.ToLower(parent.Name), "python", "node", "bash", "java", "powershell", "cmd") {
			// get first arg
			parent.Arguments = append(parent.Arguments, args[1])
		}
	}

	return
}
