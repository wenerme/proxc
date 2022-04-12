package proxc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestServer(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "proxc")
	defer os.RemoveAll(dir)
	svr := NewServer(&ServerConf{
		DBDir:   dir,
		WebAddr: ":0",
		Addr:    ":0",
	})
	if svr.Init() != nil {
		t.Fatal("Init failed")
	}
}
