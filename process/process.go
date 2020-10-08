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
	Bin                        string       `json:"-"`
	Args                       []string     `json:"-"`
	Cmd                        *exec.Cmd    `json:"-"`
	Stderr, Stdout             bytes.Buffer `json:"-"`
	StderrReader, StdoutReader io.Reader    `json:"-"`
	printMux                   sync.Mutex
}

// NewProc creates a new process and returns it
func NewProc(bin string, args ...string) (p *Proc, err error) {
	p = &Proc{Bin: bin, Args: args}
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

// Start executes the dbt command
func (p *Proc) Start() (err error) {

	p.Cmd = exec.Command(p.Bin, p.Args...)
	p.Cmd.Stderr = &p.Stderr
	p.Cmd.Stdout = &p.Stdout
	p.StdoutReader, _ = p.Cmd.StdoutPipe()
	p.StderrReader, _ = p.Cmd.StderrPipe()

	g.Trace("Proc command -> %s", p.CmdStr())
	err = p.Cmd.Start()
	if err != nil {
		err = g.Error(err, p.CmdErrorText())
	}
	return
}

// ScanWith scans with the provided function
func (p *Proc) ScanWith(stdout bool, stderr bool, scanFunc func(text string)) {
	if stdout {
		go func() {
			scanner := p.StdoutScannerLines()
			for scanner.Scan() {
				p.printMux.Lock() // print one line at a time
				scanFunc(scanner.Text())
				p.printMux.Unlock()
			}
		}()
	}
	if stderr {
		go func() {
			scanner := p.StderrScannerLines()
			for scanner.Scan() {
				p.printMux.Lock() // print one line at a time
				scanFunc(scanner.Text())
				p.printMux.Unlock()
			}
		}()
	}
}

// Run executes the dbt command, prints output and waits for it to finish
func (p *Proc) Run(printOutput bool) (err error) {
	err = p.Start()
	if err != nil {
		err = g.Error(err, "could not start process")
		return
	}

	if printOutput {
		p.ScanWith(true, true, func(t string) { fmt.Print(t) })
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
func (p *Proc) StdoutScannerLines() *bufio.Scanner {
	scanner := bufio.NewScanner(p.StdoutReader)
	scanner.Split(bufio.ScanLines)
	return scanner
}

// StderrScannerLines returns a scanner for stderr
func (p *Proc) StderrScannerLines() *bufio.Scanner {
	scanner := bufio.NewScanner(p.StderrReader)
	scanner.Split(bufio.ScanLines)
	return scanner
}
