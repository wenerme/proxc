package httpcache

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wenerme/proxc/httpencoding"
	"github.com/wenerme/wego/testx"

	"github.com/wenerme/proxc/httpcache/dbcache"
	"github.com/wenerme/proxc/httpcache/dbcache/models"
)

func TestGzip(t *testing.T) {
	resetTest()
	req := testx.Must(http.NewRequest("GET", s.server.URL+"/encoding", nil))
	// initial request without encoding
	resp := testx.Must(s.client.Do(req))
	_, _ = io.ReadAll(resp.Body)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	resp = testx.Must(s.client.Do(req))
	assert.Equal(t, resp.Header.Get("Content-Encoding"), "gzip")
	assert.Equal(t, resp.Header.Get(XFromCache), "1")
	assert.True(t, bytes.Equal(testData, testx.MustNonEOF(httpencoding.ContentEncodingReadAll(resp))))
}

func TestFile(t *testing.T) {
	resetTest()

	{
		req, err := http.NewRequest("GET", s.server.URL+"/file", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp := testx.Must(s.client.Do(req))
		_, _ = io.ReadAll(resp.Body)
		resp.Body.Close()

		resp = testx.Must(s.client.Do(req))
		assert.Equal(t, resp.Header.Get(XFromCache), "1")

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
