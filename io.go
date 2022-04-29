package g

import (
	"bufio"
	"io"
	"os"
	"runtime"
)

// UserHomeDir returns the home directory of the running user
func UserHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	} else if runtime.GOOS == "linux" {
		home := os.Getenv("XDG_CONFIG_HOME")
		if home != "" {
			return home
		}
	}
	return os.Getenv("HOME")
}

// Peek allows peeking without advancing the reader
func Peek(reader io.Reader, n int) (data []byte, readerNew io.Reader, err error) {
	bReader := bufio.NewReader(reader)
	if n == 0 {
		n = bReader.Size()
	}
	readerNew = bReader
	data, err = bReader.Peek(n)
	if err == io.EOF {
		err = nil
	} else if err != nil {
		err = Error(err, "could not Peek")
		return
	}

	return
}

// A PipeReader is the read half of a pipe.
type Pipe struct {
	Reader       *io.PipeReader
	Writer       *io.PipeWriter
	BytesRead    int64
	BytesWritten int64
}

func (p *Pipe) Read(data []byte) (n int, err error) {
	br, err := p.Reader.Read(data)
	p.BytesRead = p.BytesRead + int64(br)
	return br, err
}

func (p *Pipe) Write(data []byte) (n int, err error) {
	bw, err := p.Writer.Write(data)
	p.BytesWritten = p.BytesWritten + int64(bw)
	return bw, err
}

func (p *Pipe) close() error {
	eg := ErrorGroup{}
	eg.Capture(p.Reader.Close())
	eg.Capture(p.Writer.Close())
	return eg.Err()
}

func NewPipe() *Pipe {
	pipeR, pipeW := io.Pipe()

	pipe := &Pipe{
		Reader:       pipeR,
		Writer:       pipeW,
		BytesRead:    0,
		BytesWritten: 0,
	}

	return pipe
}
