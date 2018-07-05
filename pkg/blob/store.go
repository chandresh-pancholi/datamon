package blob

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// Store implementations know how to write entries to a K/V store.Store.
//
// Typically this is something file system-like. Examples are S3, local FS, NFS, ...
// Implementations of this interface are assumed to be fairly simple.
type Store interface {
	Get(string) (io.ReadCloser, error)
	Put(string, io.Reader) error
	Delete(string) error
	Keys() ([]string, error)
	Clear() error
}

// LocalFS creates a new local file system backed blob store
func LocalFS(baseDir string) Store {
	return &localFS{
		baseDir: filepath.Join(baseDir, "objects"),
		fs:      afero.NewOsFs(),
	}
}

type localFS struct {
	baseDir string
	fs      afero.Fs
}

func (l *localFS) fpath(key string) string {
	return filepath.Join(l.baseDir, key[:2], key[2:4], key[4:])
}

func (l *localFS) Get(key string) (io.ReadCloser, error) {
	return l.fs.Open(l.fpath(key))
}

func (l *localFS) Put(key string, rdr io.Reader) error {
	fp := l.fpath(key)
	if err := l.fs.MkdirAll(filepath.Dir(fp), 0700); err != nil {
		return fmt.Errorf("ensuring directories for %q: %v", key, err)
	}

	fi, err := l.fs.OpenFile(fp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("create record for %q: %v", key, err)
	}
	defer fi.Close()

	_, err = io.Copy(fi, rdr)
	if err != nil {
		return fmt.Errorf("write record for %q: %v", key, err)
	}

	return fi.Close()
}

func (l *localFS) Delete(key string) error {
	if err := l.fs.Remove(l.fpath(key)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %q: %v", key, err)
	}
	return nil
}

func (l *localFS) Keys() ([]string, error) {
	fis, err := afero.ReadDir(l.fs, l.fpath(""))
	if err != nil {
		return nil, err
	}

	var result []string
	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}
		result = append(result, filepath.Base(fi.Name()))
	}
	return result, nil
}

func (l *localFS) Clear() error {
	return l.fs.RemoveAll(l.baseDir)
}
