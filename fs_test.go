package vfs

import (
	"strings"
	"testing"
)

func TestFS(t *testing.T) {
	
	fs := NewFS()
	
	fs.Add("/", "1.txt", []byte("1"))
	fs.Add("/", "2.txt", []byte("2"))
	fs.Add("/a", "3.txt", []byte("3"))
	fs.Add("/a/b/c", "4.txt", []byte("4"))
	
	t.Log(len(fs.Paths()))
	for k, v := range fs.Paths() {
		switch r := v.(type) {
		case *DirInfo:
			var list []string
			for _, e := range r.entries {
				list = append(list, e.Name())
			}
			t.Log("DirInfo", k, strings.Join(list, ","))
		case *CompressedFileInfo:
			t.Log("CompressedFileInfo", k, string(r.compressedContent))
		}
	}
}
