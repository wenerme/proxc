package httpcache

import (
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/wenerme/proxc/httpcache/dbcache"
	"github.com/wenerme/proxc/httpcache/dbcache/models"
)

func TestFile(t *testing.T) {
	resetTest()
	{
		req, err := http.NewRequest("GET", s.server.URL+"/file", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := s.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		hash := models.ContentHashBytes(bytes)
		file, err := os.ReadFile("httpcache_test.go")
		if err != nil {
			t.Fatal(err)
		}
		h2 := models.ContentHashBytes(file)

		if hash != h2 {
			t.Fatalf("hash not equal: expected %s, got %s", h2, hash)
		}

		{
			_, fdb, _ := s.transport.Cache.(*dbcache.Cache).GetDB(req)
			fc := &models.FileContent{}
			if fdb.Where(models.FileContent{Hash: hash}).First(fc).Error != nil {
				t.Fatal("file not found")
			}
			if fc.Hash != hash {
				t.Fatalf("hash not equal: expected %s, got %s", hash, fc.Hash)
			}
			fr := &models.FileRef{}
			if fdb.Where(models.FileRef{URL: req.URL.String()}).First(fr).Error != nil {
				t.Fatal("file ref not found")
			}
			if fr.Hash != hash {
				t.Fatalf("hash not equal: expected %s, got %s", hash, fr.Hash)
			}
		}
	}
}
