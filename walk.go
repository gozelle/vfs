package vfs

import (
	fsi "io/fs"
	"net/http"
	"path/filepath"
	"sort"
)

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
