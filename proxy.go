package vfs

import (
	"errors"
	"fmt"
	fsi "io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func Proxy(dir string) http.FileSystem {
	var err error
	defer func() {
		if err != nil {
			panic(fmt.Errorf("new bundle error: %s", err))
		}
	}()
	
	p := dir
	if !strings.HasPrefix(dir, "/") {
		var pwd string
		pwd, err = os.Getwd()
		if err != nil {
			return nil
		}
		p = filepath.Join(pwd, dir)
	}
	
	info, err := os.Stat(p)
	if err != nil {
		return nil
	}
	if !info.IsDir() {
		err = fmt.Errorf("bundle path is not dir: %s", p)
		return nil
	}
	return Dir(p)
}

type Dir string

// mapDirOpenError maps the provided non-nil error from opening name
// to a possibly better non-nil error. In particular, it turns OS-specific errors
// about opening files in non-directories into os.ErrNotExist. See Issue 18984.
func mapDirOpenError(originalErr error, name string) error {
	if os.IsNotExist(originalErr) || os.IsPermission(originalErr) {
		return originalErr
	}
	
	parts := strings.Split(name, string(filepath.Separator))
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		fi, err := os.Stat(strings.Join(parts[:i+1], string(filepath.Separator)))
		if err != nil {
			return originalErr
		}
		if !fi.IsDir() {
			return os.ErrNotExist
		}
	}
	return originalErr
}

// Open implements FileSystem using os.Open, opening files for reading rooted
// and relative to the directory d.
func (p Dir) Open(name string) (http.File, error) {
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return nil, errors.New("http: invalid character in file path")
	}
	
	fullPath := p.prepare(name)
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, mapDirOpenError(err, fullPath)
	}
	return &File{f: f}, nil
}

func (p Dir) prepare(name string) string {
	dir := string(p)
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, filepath.FromSlash(path.Clean("/"+name)))
}

var _ http.File = (*File)(nil)

type File struct {
	f http.File
}

func (p File) Close() error {
	return p.f.Close()
}

func (p File) Read(len []byte) (int, error) {
	return p.f.Read(len)
}

func (p File) Seek(offset int64, whence int) (int64, error) {
	return p.f.Seek(offset, whence)
}

func (p File) Readdir(count int) ([]os.FileInfo, error) {
	infos, err := p.f.Readdir(count)
	if err != nil {
		return infos, err
	}
	var res []os.FileInfo
	for _, v := range infos {
		if v.IsDir() {
			res = append(res, v)
		}
	}
	return res, nil
}

func (p File) Stat() (os.FileInfo, error) {
	return p.f.Stat()
}

var SkipDir error = fsi.SkipDir

type WalkFunc func(path string, info fsi.FileInfo, err error) error

func Walk(fs http.FileSystem, root string, fn WalkFunc) (err error) {
	f, err := fs.Open(root)
	if err != nil {
		return
	}
	defer func() {
		_ = f.Close()
	}()
	
	info, err := f.Stat()
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = walk(fs, root, info, fn)
	}
	
	if err == SkipDir {
		err = nil
		return
	}
	
	return
}

// walk recursively descends path, calling walkFn.
func walk(fs http.FileSystem, path string, info fsi.FileInfo, walkFn WalkFunc) error {
	if !info.IsDir() {
		return walkFn(path, info, nil)
	}
	
	names, err := readDirNames(fs, path)
	err1 := walkFn(path, info, err)
	// If err != nil, walk can't walk into this directory.
	// err1 != nil means walkFn want walk to skip this directory or stop walking.
	// Therefore, if one of err and err1 isn't nil, walk will return.
	if err != nil || err1 != nil {
		// The caller's behavior is controlled by the return value, which is decided
		// by walkFn. walkFn may ignore err and return nil.
		// If walkFn returns SkipDir, it will be handled by the caller.
		// So walk should return whatever walkFn returns.
		return err1
	}
	
	for _, name := range names {
		filename := filepath.Join(path, name)
		var i fsi.FileInfo
		i, err = stat(fs, filename)
		if err != nil {
			if err = walkFn(filename, i, err); err != nil && err != SkipDir {
				return err
			}
		} else {
			err = walk(fs, filename, i, walkFn)
			if err != nil {
				if !i.IsDir() || err != SkipDir {
					return err
				}
			}
		}
	}
	return nil
}

func stat(fs http.FileSystem, path string) (fsi.FileInfo, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	
	return f.Stat()
}

func readDirNames(fs http.FileSystem, dirname string) ([]string, error) {
	f, err := fs.Open(dirname)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	infos, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}
	
	var names []string
	for _, v := range infos {
		names = append(names, v.Name())
	}
	
	sort.Strings(names)
	return names, nil
}
