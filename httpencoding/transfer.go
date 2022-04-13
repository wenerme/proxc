package httpencoding

import (
	"bytes"
	"io"
	"strings"

	"github.com/pkg/errors"
)

func AcceptEncoding(from string, to string) (choose string, err error) {
	if from == EncodingIdentity {
		from = ""
	}
	if to == EncodingIdentity {
		to = ""
	}
	if from == to {
		return to, nil
	}
	s := strings.Split(to, ",")
	for _, choose = range s {
		choose = strings.TrimSpace(choose)
		switch {
		case choose == "":
			continue
		case choose == from:
			return
		case IsSupported(choose):
			return
		}
	}
	err = errors.Errorf("unsupported encoding: %s", to)
	return
}

func Transfer(from string, in io.Reader, to string, out io.Writer) (int64, error) {
	if in == nil {
		return 0, nil
	}
	if from == "" {
		from = EncodingIdentity
	}
	if to == "" {
		to = EncodingIdentity
	}
	if from == to {
		return io.Copy(out, in)
	}

	c1 := codecs[from]
	c2 := codecs[to]
	if c1 == nil || c2 == nil {
		return 0, errors.Errorf("unsupported encoding %q -> %q", from, to)
	}
	r, err := c1.NewReader(in)
	if err != nil {
		return 0, err
	}
	w, err := c2.NewWriter(out)
	if err != nil {
		return 0, err
	}
	defer w.Close()
	// defer r.Close()
	return io.Copy(w, r)
}

func TransferBytes(from string, in []byte, to string) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	_, err := Transfer(from, bytes.NewReader(in), to, buf)
	return buf.Bytes(), err
}
