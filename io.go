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
