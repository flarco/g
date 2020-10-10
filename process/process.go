package process

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	g "github.com/flarco/gutil"
)

// Proc is a command background process
type Proc struct {
	Bin                        string
	Args                       []string
	Cmd                        *exec.Cmd
	Workdir                    string
	Stderr, Stdout             bytes.Buffer
	StderrReader, StdoutReader io.Reader
	printMux                   sync.Mutex
	scanner                    *scanConfig
}

type scanConfig struct {
	stdout   bool
	stderr   bool
	scanFunc func(src, text string)
}

// NewProc creates a new process and returns it
func NewProc(bin string, args ...string) (p *Proc, err error) {
	p = &Proc{
		Bin:  bin,
		Args: args,
	}

	if !p.ExecutableFound() {
		err = g.Error(fmt.Errorf("executable '%s' not found in path", p.Bin))
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
	if p.scanner != nil {
		p.StdoutReader, err = p.Cmd.StdoutPipe()
		if err != nil {
			return g.Error(err)
		}
		p.StderrReader, err = p.Cmd.StderrPipe()
		if err != nil {
			return g.Error(err)
		}
	} else {
		p.Stdout.Reset()
		p.Stderr.Reset()
		p.Cmd.Stderr = &p.Stderr
		p.Cmd.Stdout = &p.Stdout
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
func (p *Proc) SetScanner(stdout bool, stderr bool, scanFunc func(src, text string)) {
	p.scanner = &scanConfig{stdout: stdout, stderr: stderr, scanFunc: scanFunc}
}

// SetPrint set scanner to print
func (p *Proc) SetPrint() {
	p.SetScanner(true, true, func(s, t string) { fmt.Println(t) })
}

func (p *Proc) scan() {
	if p.scanner == nil {
		return
	}

	if p.scanner.stdout {
		go func() {
			scanner := p.StdoutScannerLines()
			if scanner == nil {
				return
			}
			for scanner.Scan() {
				p.printMux.Lock() // print one line at a time
				p.scanner.scanFunc("stdout", scanner.Text())
				p.printMux.Unlock()
			}
		}()
	}
	if p.scanner.stderr {
		go func() {
			scanner := p.StderrScannerLines()
			if scanner == nil {
				return
			}
			for scanner.Scan() {
				p.printMux.Lock() // print one line at a time
				p.scanner.scanFunc("stderr", scanner.Text())
				p.printMux.Unlock()
			}
		}()
	}
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
	}
	return
}

// CmdStr returns the command string
func (p *Proc) CmdStr() string {
	return strings.Join(append([]string{p.Bin}, p.Args...), " ")
}

// CmdErrorText returns the command error text
func (p *Proc) CmdErrorText() string {
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
