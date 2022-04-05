package g

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

// FileItem represents a file or a directory
type FileItem struct {
	Name       string
	RelPath    string // relative path
	FullPath   string
	ParentPath string
	IsDir      bool
	ModTs      int64
	Size       int64
}

// ListDir lists the file in given directory path recursively
func ListDir(dirPath string) (files []FileItem, err error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		err = Error(err, "unable to read directory %s", dirPath)
		return
	}

	for _, entry := range entries {
		info, _ := entry.Info()
		fullpath := filepath.Clean(path.Join(dirPath, entry.Name()))
		file := FileItem{
			Name:       entry.Name(),
			RelPath:    entry.Name(),
			FullPath:   fullpath,
			ParentPath: filepath.Dir(fullpath),
			IsDir:      entry.IsDir(),
			ModTs:      info.ModTime().Unix(),
			Size:       info.Size(),
		}
		file.RelPath = strings.TrimPrefix(file.RelPath, "./")
		file.RelPath = strings.TrimPrefix(file.RelPath, "/")
		file.RelPath = strings.TrimPrefix(file.RelPath, `\`)
		files = append(files, file)
	}
	return
}

// ListDirRecursive lists the file in given directory path recursively
func ListDirRecursive(dirPath string) (files []FileItem, err error) {

	walkFunc := func(subPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if subPath == dirPath {
			return nil
		}

		subPath = filepath.Clean(subPath)
		file := FileItem{
			Name:       info.Name(),
			RelPath:    strings.TrimPrefix(subPath, dirPath),
			FullPath:   subPath,
			ParentPath: filepath.Dir(subPath),
			IsDir:      info.IsDir(),
			ModTs:      info.ModTime().Unix(),
			Size:       info.Size(),
		}

		file.RelPath = strings.TrimPrefix(file.RelPath, "./")
		file.RelPath = strings.TrimPrefix(file.RelPath, "/")
		file.RelPath = strings.TrimPrefix(file.RelPath, `\`)
		files = append(files, file)
		return nil
	}
	err = filepath.Walk(dirPath, walkFunc)
	if err != nil {
		err = Error(err, "Error listing "+dirPath)
	}
	return
}
