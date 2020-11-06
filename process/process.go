package process

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	g "github.com/flarco/g"
)

// Session is a session to execute processes and keep output
type Session struct {
	Proc           *Proc
	Alias          map[string]string
	Workdir        string
	Print          bool
	Stderr, Stdout string
	scanner        *scanConfig
	mux            sync.Mutex
}

// Proc is a command background process
type Proc struct {
	Bin                        string
	Args                       []string
	Cmd                        *exec.Cmd
	Workdir                    string
	Print                      bool
	HideCmdInErr               bool
	Stderr, Stdout             bytes.Buffer
	StderrReader, StdoutReader io.Reader
	printMux                   sync.Mutex
	scanner                    *scanConfig
}

type scanConfig struct {
	scanFunc func(stderr bool, text string)
}

// NewSession creates a session to execute processes
func NewSession() (s *Session) {
	s = &Session{
		Alias: map[string]string{},
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
	s.scanner = &scanConfig{scanFunc: scanFunc}
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
	p.Workdir = s.Workdir
	p.Print = s.Print
	p.scanner = s.scanner

	err = p.Run()
	if err != nil {
		e, ok := err.(*g.ErrType)
		if ok {
			// replace alias value with alias key in messages
			// this is to hide unneeded/unwanted details
			for k, v := range s.Alias {
				for i, msg := range e.MsgStack {
					e.MsgStack[i] = strings.ReplaceAll(msg, v, k)
				}
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
		p.CmdStr(),
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
		Bin:  bin,
		Args: args,
	}

	if !p.ExecutableFound() {
		err = g.Error("executable '%s' not found in path", p.Bin)
	}
	return
}

// ExecutableFound returns true if the executable is found
func (p *Proc) ExecutableFound() bool {
	_, err := exec.LookPath(p.Bin)
	if err != nil {
		return false
	}
	return true
}

// SetArgs sets the args for the command
func (p *Proc) SetArgs(args ...string) {
	p.Args = args
}

// Start executes the command
func (p *Proc) Start(args ...string) (err error) {
	if len(args) > 0 {
		p.SetArgs(args...)
	}
	p.Cmd = exec.Command(p.Bin, p.Args...)
	p.Cmd.Dir = p.Workdir
	p.Stdout.Reset()
	p.Stderr.Reset()

	p.StdoutReader, err = p.Cmd.StdoutPipe()
	if err != nil {
		return g.Error(err)
	}
	p.StderrReader, err = p.Cmd.StderrPipe()
	if err != nil {
		return g.Error(err)
	}

	g.Trace("Proc command -> %s", p.CmdStr())
	err = p.Cmd.Start()
	if err != nil {
		err = g.Error(err, p.CmdErrorText())
	}

	p.scan()

	return
}

// SetScanner sets scanner with the provided function
func (p *Proc) SetScanner(scanFunc func(stderr bool, text string)) {
	p.scanner = &scanConfig{scanFunc: scanFunc}
}

func (p *Proc) scan() {

	go func() {
		scanner := p.StdoutScannerLines()
		if scanner == nil {
			return
		}
		for scanner.Scan() {
			p.printMux.Lock() // print one line at a time
			text := scanner.Text()
			p.Stdout.WriteString(text + "\n")
			if p.Print {
				fmt.Fprintf(os.Stdout, "%s\n", text)
			}
			if p.scanner != nil {
				p.scanner.scanFunc(false, text)
			}
			p.printMux.Unlock()
		}
	}()

	go func() {
		scanner := p.StderrScannerLines()
		if scanner == nil {
			return
		}
		for scanner.Scan() {
			p.printMux.Lock() // print one line at a time
			text := scanner.Text()
			p.Stderr.WriteString(text + "\n")
			if p.Print {
				fmt.Fprintf(os.Stderr, "%s\n", text)
			}
			if p.scanner != nil {
				p.scanner.scanFunc(true, text)
			}
			p.printMux.Unlock()
		}
	}()
}

// Run executes the dbt command, prints output and waits for it to finish
func (p *Proc) Run(args ...string) (err error) {
	err = p.Start(args...)
	if err != nil {
		err = g.Error(err, "could not start process. %s", p.CmdErrorText())
		return
	}

	err = p.Cmd.Wait()
	if err != nil {
		err = g.Error(err, p.CmdErrorText())
		return
	}

	if code := p.Cmd.ProcessState.ExitCode(); code != 0 {
		err = g.Error("exit code = %d. %s", code, p.CmdErrorText())
	}

	return
}

// CmdStr returns the command string
func (p *Proc) CmdStr() string {
	return strings.Join(append([]string{p.Bin}, p.Args...), " ")
}

// CmdErrorText returns the command error text
func (p *Proc) CmdErrorText() string {
	if p.HideCmdInErr {
		return g.F("%s\n%s", p.Stderr.String(), p.Stdout.String())
	}
	return fmt.Sprintf(
		"Proc command -> %s\n%s\n%s",
		p.CmdStr(), p.Stderr.String(), p.Stdout.String(),
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
	return p.Cmd.Wait()
}
