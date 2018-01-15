package targz

import (
	"os"
	"regexp"
	"strings"

	"github.com/Stratoscale/logserver/filesystem"
)

var (
	reContains = regexp.MustCompile(`\.tar(\.gz)?`)
	reSuffix   = regexp.MustCompile(`\.tar(\.gz)?$`)
)

func New(inner filesystem.FileSystem) filesystem.FileSystem {
	return &wrap{
		inner:    inner,
		tarCache: make(map[string]filesystem.FileSystem),
	}
}

type wrap struct {
	inner    filesystem.FileSystem
	tarCache map[string]filesystem.FileSystem
}

func (w *wrap) ReadDir(dirname string) ([]os.FileInfo, error) {
	tfs, innerPath, err := w.getTfs(dirname)
	if err != nil {
		return nil, err
	}
	if tfs == nil {
		files, err := w.inner.ReadDir(dirname)
		if err != nil {
			return nil, err
		}
		return changeTarToDir(files...), nil
	}
	return tfs.ReadDir(innerPath)
}

func (w *wrap) Lstat(name string) (os.FileInfo, error) {
	tfs, innerPath, err := w.getTfs(name)
	if err != nil {
		return nil, err
	}
	if tfs == nil {
		file, err := w.inner.Lstat(name)
		if err != nil {
			return nil, err
		}
		return changeTarToDir(file)[0], nil
	}
	return tfs.Lstat(innerPath)
}

func (w *wrap) Join(elem ...string) string {
	return w.inner.Join(elem...)
}

func (w *wrap) Open(name string) (filesystem.File, error) {
	tfs, innerPath, err := w.getTfs(name)
	if err != nil {
		return nil, err
	}
	if tfs == nil {
		return w.inner.Open(name)
	}
	return tfs.Open(innerPath)
}

func (w *wrap) Close() error {
	return w.inner.Close()
}

func (w *wrap) getTfs(dirname string) (filesystem.FileSystem, string, error) {
	tarName, innerPath := split(dirname)
	if tarName == "" {
		return nil, dirname, nil
	}
	if w.tarCache[tarName] == nil {
		f, err := w.inner.Open(tarName)
		if err != nil {
			return nil, "", err
		}
		w.tarCache[tarName], err = NewFS(f)
		if err != nil {
			return nil, "", err
		}
	}
	return w.tarCache[tarName], innerPath, nil
}

func split(dirname string) (tarName string, innerPath string) {
	loc := reContains.FindStringIndex(dirname)
	if len(loc) == 0 {
		return "", dirname
	}
	end := loc[1]

	tarName = dirname[:end]
	innerPath = strings.Trim(dirname[end:], sep)
	return
}

// changeTarToDir exposes tar files as directories
func changeTarToDir(files ...os.FileInfo) []os.FileInfo {
	for i, file := range files {
		if reSuffix.MatchString(file.Name()) {
			files[i] = &tarFile{FileInfo: file}
		}
	}
	return files
}

type tarFile struct{ os.FileInfo }

func (d *tarFile) IsDir() bool { return true }