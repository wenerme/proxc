package httpencoding

import (
	"bytes"
	"io"
	"net/http"
)

// AcceptEncodingWriter will handle request Accept-Encoding header, set corresponding Content-Encoding header.
// Caller must call Close() on the returned writer to flush the data.
func AcceptEncodingWriter(w http.ResponseWriter, r *http.Request) io.WriteCloser {
	enc, _ := AcceptEncoding("", r.Header.Get("Accept-Encoding"))
	if enc != "" {
		w.Header().Set("Content-Encoding", enc)
		wr, _ := NewWriter(enc, w)
		return wr
	}
	return writeCloser(w)
}

func ContentEncodingReadAll(resp *http.Response) ([]byte, error) {
	r, err := ContentEncodingReader(resp)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, r)
	return buf.Bytes(), err
}

func ContentEncodingReader(resp *http.Response) (r io.Reader, err error) {
	enc := resp.Header.Get("Content-Encoding")
	if enc != "" && !resp.Uncompressed {
		r, err = NewReader(enc, resp.Body)
		return
	}
	return resp.Body, nil
}
