package httpencoding

import (
	"bytes"
	"io"
)

type Encoding struct {
	Name      string
	NewReader func(r io.Reader) (io.ReadCloser, error)
	NewWriter func(w io.Writer) (io.WriteCloser, error)
}

func (c *Encoding) DecodeBytes(in []byte) (out []byte, err error) {
	reader, err := c.NewReader(bytes.NewReader(in))
	if err != nil {
		return
	}
	buf := bytes.Buffer{}
	_, err = io.Copy(&buf, reader)
	_ = reader.Close()
	return buf.Bytes(), err
}

func (c *Encoding) EncodeBytes(in []byte) (out []byte, err error) {
	buf := bytes.Buffer{}
	writer, err := c.NewWriter(&buf)
	if err != nil {
		return
	}
	_, err = writer.Write(in)
	_ = writer.Close()
	return buf.Bytes(), err
}
