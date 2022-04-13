package httpencoding

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wenerme/wego/testx"
)

func TestTransfer(t *testing.T) {
	raw := testx.Must(os.ReadFile("encoding.go"))

	a := testx.Must(TransferBytes("", raw, EncodingGzip))
	b := testx.Must(TransferBytes(EncodingGzip, a, EncodingBrotli))
	c := testx.Must(TransferBytes(EncodingBrotli, b, EncodingZstd))
	r := testx.Must(TransferBytes(EncodingZstd, c, ""))

	assert.True(t, bytes.Equal(raw, r))
}
