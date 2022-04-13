package httpencoding

import (
	"compress/gzip"
	"compress/zlib"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"github.com/pkg/errors"
)

const (
	EncodingZstd     = "zstd"
	EncodingIdentity = "identity"
	EncodingDeflate  = "deflate"
	EncodingGzip     = "gzip"
	EncodingBrotli   = "br"
)

func NewWriter(enc string, w io.Writer) (out io.WriteCloser, err error) {
	if c := codecs[enc]; c != nil {
		return c.NewWriter(w)
	}
	return nil, errors.Errorf("unsupported encoding: %s", enc)
}

func NewReader(enc string, r io.Reader) (io.ReadCloser, error) {
	if c := codecs[enc]; c != nil {
		return c.NewReader(r)
	}
	return nil, errors.Errorf("unsupported encoding: %s", enc)
}

var codecs = make(map[string]*Encoding, 6)

func RegisterEncoding(name string, c *Encoding) {
	codecs[name] = c
}

func IsSupported(name string) bool {
	_, ok := codecs[name]
	return ok
}

func init() {
	identity := &Encoding{
		Name: EncodingIdentity,
		NewReader: func(r io.Reader) (io.ReadCloser, error) {
			return readCloser(r), nil
		},
		NewWriter: func(w io.Writer) (io.WriteCloser, error) {
			return writeCloser(w), nil
		},
	}
	RegisterEncoding("", identity)
	RegisterEncoding(EncodingIdentity, identity)
	RegisterEncoding(EncodingGzip, &Encoding{
		Name: EncodingGzip,
		NewWriter: func(w io.Writer) (io.WriteCloser, error) {
			return gzip.NewWriter(w), nil
		},
		NewReader: func(r io.Reader) (io.ReadCloser, error) {
			return gzip.NewReader(r)
		},
	})
	RegisterEncoding(EncodingDeflate, &Encoding{
		Name: EncodingDeflate,
		NewWriter: func(w io.Writer) (io.WriteCloser, error) {
			return zlib.NewWriter(w), nil
		},
		NewReader: zlib.NewReader,
	})
	RegisterEncoding(EncodingBrotli, &Encoding{
		Name: EncodingBrotli,
		NewReader: func(r io.Reader) (io.ReadCloser, error) {
			return io.NopCloser(brotli.NewReader(r)), nil
		},
		NewWriter: func(w io.Writer) (io.WriteCloser, error) {
			return brotli.NewWriter(w), nil
		},
	})
	RegisterEncoding(EncodingZstd, &Encoding{
		Name: EncodingZstd,
		NewWriter: func(w io.Writer) (io.WriteCloser, error) {
			return zstd.NewWriter(w)
		},
		NewReader: func(r io.Reader) (io.ReadCloser, error) {
			zr, err := zstd.NewReader(r)
			if err != nil {
				return nil, err
			}
			return zr.IOReadCloser(), nil
		},
	})
}
