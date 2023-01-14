package vfs

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func Proxy(dir string) http.FileSystem {
	var err error
	defer func() {
		if err != nil {
			panic(fmt.Errorf("proxy error: %s", err))
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
		err = fmt.Errorf("proxy path: %s is not dir", p)
		return nil
	}
	return http.Dir(p)
}
