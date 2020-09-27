package gutil

import (
	"bufio"
	"github.com/markbates/pkger"
	"github.com/markbates/pkger/pkging"
	"io"
	"io/ioutil"
	"os"
	"path"
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

// PkgerString returns the packager file string
func PkgerString(name string) (content string, err error) {
	_, filename, _, _ := runtime.Caller(1)
	file, err := pkger.Open(path.Join(path.Dir(filename), name))
	if err != nil {
		return "", Error(err, "could not open pker file: %s", name)
	}

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", Error(err, "could not read pker file: ", name)
	}
	return string(fileBytes), nil
}

// PkgerFile returns the packager file
func PkgerFile(name string) (file pkging.File, err error) {
	_, filename, _, _ := runtime.Caller(1)
	TemplateFile, err := pkger.Open(path.Join(path.Dir(filename), name))
	if err != nil {
		return nil, Error(err, "cannot open "+path.Join(path.Dir(filename), name))
	}
	return TemplateFile, nil
}

