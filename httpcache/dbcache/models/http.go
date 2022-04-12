package models

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"

	"github.com/klauspost/compress/zstd"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

type HTTPResponse struct {
	Model
	// Key       string //
	Method string `gorm:"uniqueIndex:idx_http_responses_method_url"`
	URL    string `gorm:"uniqueIndex:idx_http_responses_method_url"`
	Host   string
	Path   string

	// Size int
	// Raw  []byte

	Proto       string
	StatusCode  int
	Header      datatypes.JSON
	Encoding    string // gzip, deflate, br, zstd, identity
	RawSize     int64  // uncompressed size
	BodySize    int64
	Body        []byte
	ContentType string
	ContentHash string // sha2-256
	FileName    string
}

func (HTTPResponse) ConflictColumns() []clause.Column {
	return []clause.Column{{Name: "method"}, {Name: "url"}}
}

func ContentHashBytes(v []byte) string {
	sum := sha256.Sum256(v)
	return hex.EncodeToString(sum[:])
}

func ContentHash(r io.Reader) (string, error) {
	sum := sha256.New()
	if _, err := io.Copy(sum, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func EncodingWriter(enc string, in io.Writer) (out io.Writer, err error) {
	switch enc {
	case "gzip":
		return gzip.NewWriter(in), nil
	case "deflate":
		return zlib.NewWriter(in), nil
	case "zstd":
		return zstd.NewWriter(in)
	case "identity", "":
		return in, nil
	default:
		return nil, errors.Errorf("unknown encoding: %s", enc)
	}
}

func closeCloser(v interface{}) error {
	if v == nil {
		return nil
	}
	if c, ok := v.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func EncodingReader(enc string, r io.Reader) (io.ReadCloser, error) {
	switch enc {
	case "gzip":
		return gzip.NewReader(r)
	case "deflate":
		return zlib.NewReader(r)
	case "zstd":
		d, err := zstd.NewReader(r)
		if err != nil {
			return nil, err
		}
		return d.IOReadCloser(), err
	case "identity", "":
		return readCloser(r), nil
	default:
		return nil, errors.Errorf("unknown encoding: %s", enc)
	}
}

func (m *HTTPResponse) ReadAll() (out []byte, err error) {
	body, err := m.GetBody()
	if err != nil {
		return nil, err
	}
	defer body.Close()

	return io.ReadAll(body)
}

func (m *HTTPResponse) GetBody() (rc io.ReadCloser, err error) {
	if len(m.Body) == 0 {
		return http.NoBody, nil
	}
	var r io.Reader = bytes.NewReader(m.Body)
	r, err = EncodingReader(m.Encoding, r)
	return readCloser(r), err
}

func readCloser(r io.Reader) io.ReadCloser {
	if r == nil {
		return nil
	}
	if rc, _ := r.(io.ReadCloser); rc != nil {
		return rc
	}
	return io.NopCloser(r)
}

func (m *HTTPResponse) SetResponse(resp *http.Response) (err error) {
	res := resp.Request
	m.Method = res.Method
	m.URL = res.URL.String()
	m.Host = res.URL.Host
	m.Path = res.URL.Path

	m.Proto = resp.Proto
	m.StatusCode = resp.StatusCode
	m.Header, err = json.Marshal(resp.Header)
	if err != nil {
		return err
	}
	m.ContentType, _, _ = mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if m.Encoding == "" && shouldCompress[m.ContentType] {
		m.Encoding = "zstd"
	}

	if hdr := resp.Header.Get("Content-Disposition"); hdr != "" {
		_, params, _ := mime.ParseMediaType(hdr)
		filename := params["filename"]
		if filename != "" {
			m.FileName = filename
		}
	}

	var body []byte
	body, resp.Body, err = drainBody(resp.Body)
	if err != nil {
		return errors.Wrap(err, "drain body")
	}

	switch {
	case m.Encoding == "" || m.Encoding == "identity":
		m.Body = body
		m.BodySize = int64(len(m.Body))
		m.RawSize = m.BodySize
	default:
		buf := &bytes.Buffer{}
		var w io.Writer = buf

		w, err = EncodingWriter(m.Encoding, w)
		if err == nil {
			_, err = w.Write(body)
			err = multierr.Combine(err, closeCloser(w))
		}
		if err == nil {
			m.Body = buf.Bytes()
		}
		m.RawSize = int64(len(body))
		m.BodySize = int64(len(m.Body))
	}
	if err != nil {
		return
	}

	if m.FileName != "" {
		m.ContentHash = ContentHashBytes(body)
	}

	return
}

func (m *HTTPResponse) GetResponse(req *http.Request) (resp *http.Response, err error) {
	if req == nil {
		req = &http.Request{
			Method: "GET",
		}
		req.URL, err = url.Parse(m.URL)
		if err != nil {
			return
		}
	}

	//if len(m.Raw) > 0 {
	//	return http.ReadResponse(bufio.NewReader(bytes.NewReader(m.Raw)), req)
	//}

	resp = &http.Response{
		StatusCode:    m.StatusCode,
		Proto:         m.Proto,
		Request:       req,
		ContentLength: m.BodySize,
	}
	resp.ProtoMajor, resp.ProtoMinor, _ = http.ParseHTTPVersion(m.Proto)
	if s := http.StatusText(m.StatusCode); s != "" {
		resp.Status = fmt.Sprintf("%d %s", m.StatusCode, s)
	}

	err = json.Unmarshal(m.Header, &resp.Header)
	if err != nil {
		return
	}
	resp.Body, err = m.GetBody()
	return
}

func drainBody(b io.ReadCloser) (body []byte, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return nil, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	body = buf.Bytes()
	return body, io.NopCloser(bytes.NewReader(body)), nil
}

var shouldCompress = map[string]bool{}

func init() {
	// https://support.cloudflare.com/hc/en-us/articles/200168396
	for _, v := range []string{
		"text/html",
		"text/richtext",
		"text/plain",
		"text/css",
		"text/x-script",
		"text/x-component",
		"text/x-java-source",
		"text/x-markdown",
		"application/javascript",
		"application/x-javascript",
		"text/javascript",
		"text/js",
		"image/x-icon",
		"image/vnd.microsoft.icon",
		"application/x-perl",
		"application/x-httpd-cgi",
		"text/xml",
		"application/xml",
		"application/xml+rss",
		"application/vnd.api+json",
		"application/x-protobuf",
		"application/json",
		"multipart/bag",
		"multipart/mixed",
		"application/xhtml+xml",
		"font/ttf",
		"font/otf",
		"font/x-woff",
		"image/svg+xml",
		"application/vnd.ms-fontobject",
		"application/ttf",
		"application/x-ttf",
		"application/otf",
		"application/x-otf",
		"application/truetype",
		"application/opentype",
		"application/x-opentype",
		"application/font-woff",
		"application/eot",
		"application/font",
		"application/font-sfnt",
		"application/wasm",
		"application/javascript-binast",
		"application/manifest+json",
		"application/ld+json",
	} {
		shouldCompress[v] = true
	}
}
