package httpencoding

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/wenerme/wego/testx"
)

func TestEncoding(t *testing.T) {
	raw := testx.Must(os.ReadFile("encoding.go"))
	for k, c := range codecs {
		out := testx.Must(c.EncodeBytes(raw))
		out2 := testx.Must(c.DecodeBytes(out))
		if !bytes.Equal(raw, out2) {
			t.Fatal("mismatch", k)
		}
		t.Logf("%s: %d -> %d %.2f", k, len(raw), len(out), float64(len(raw))/float64(len(out)))
	}
}

func TestServer(t *testing.T) {
	mux := http.NewServeMux()
	raw := testx.Must(os.ReadFile("encoding.go"))

	mux.HandleFunc("/encoding", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		writer := AcceptEncodingWriter(w, r)
		testx.Must(writer.Write(raw))
		testx.NoErr(writer.Close())
	})

	svr := httptest.NewServer(mux)
	defer svr.Close()
	req := testx.Must(http.NewRequest("GET", svr.URL+"/encoding", nil))
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	resp := testx.Must(http.DefaultClient.Do(req))
	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
	assert.True(t, bytes.Equal(raw, MustNonEOF(ContentEncodingReadAll(resp))))
	testx.NoErr(resp.Body.Close())
}

func MustNonEOF[T any](v T, err error) T {
	if err != nil && !errors.Is(err, io.EOF) {
		panic(err)
	}
	return v
}
