package models

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"

	"github.com/wenerme/proxc/httpencoding"

	"github.com/pkg/errors"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

type HTTPResponse struct {
	Model
	Method string `gorm:"uniqueIndex:idx_http_responses_method_url"`
	URL    string `gorm:"uniqueIndex:idx_http_responses_method_url"`
	Host   string
	Path   string

	// Size int
	// Raw  []byte

	Proto           string
	StatusCode      int
	Header          datatypes.JSON
	RawSize         int64 // size before encoding
	BodySize        int64 // size of Body
	Body            []byte
	ContentType     string
	ContentEncoding string // gzip, deflate, br, zstd, identity
	ContentHash     string // sha2-256 for raw data for file
	FileName        string
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

var DefaultEncoding = httpencoding.EncodingZstd

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
	return httpencoding.NewReader(m.ContentEncoding, bytes.NewReader(m.Body))
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

	bodyEncoding := resp.Header.Get("Content-Encoding")
	if resp.Uncompressed {
		bodyEncoding = ""
	}
	// reduce an encoding process
	m.ContentEncoding = bodyEncoding

	if m.ContentEncoding == "" && shouldCompress[m.ContentType] {
		m.ContentEncoding = DefaultEncoding
	}
	var body io.Reader
	body, resp.Body, err = drainBody(resp.Body)
	if err != nil {
		return errors.Wrap(err, "drain body")
	}
	buf := bytes.NewBuffer(nil)
	m.RawSize, err = httpencoding.Transfer(bodyEncoding, body, m.ContentEncoding, buf)
	if err != nil {
		return errors.Wrap(err, "transfer body")
	}
	m.Body = buf.Bytes()
	m.BodySize = int64(len(m.Body))

	if hdr := resp.Header.Get("Content-Disposition"); hdr != "" {
		_, params, _ := mime.ParseMediaType(hdr)
		filename := params["filename"]
		if filename != "" {
			m.FileName = filename
		}
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

	enc, _ := httpencoding.AcceptEncoding(m.ContentEncoding, req.Header.Get("Accept-Encoding"))
	if enc == "" {
		resp.Body, err = m.GetBody()
		resp.Header.Del("Content-Encoding")
		resp.Header.Del("Content-Length")
	} else {
		buf := bytes.NewBuffer(nil)
		_, err = httpencoding.Transfer(m.ContentEncoding, bytes.NewReader(m.Body), enc, buf)
		resp.Body = io.NopCloser(buf)
		resp.Header.Set("Content-Encoding", enc)
		resp.Header.Del("Content-Length")
	}

	return
}

func drainBody(b io.ReadCloser) (r1 io.ReadCloser, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return io.NopCloser(&buf), io.NopCloser(bytes.NewReader(buf.Bytes())), nil
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
