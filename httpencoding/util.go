package httpencoding

import "io"

func writeCloser(r io.Writer) io.WriteCloser {
	if r == nil {
		return nil
	}
	if rc, _ := r.(io.WriteCloser); rc != nil {
		return rc
	}
	return nopCloser{r}
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

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }
