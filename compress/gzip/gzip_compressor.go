package gzip

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/JrMarcco/easy-rpc/compress"
)

var _ compress.Compressor = (*Compressor)(nil)

type Compressor struct{}

func (c *Compressor) Code() uint8 {
	return compress.CompressorGzip
}

func (c *Compressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	_, err := writer.Write(data)
	if err != nil {
		return nil, err
	}

	// writer 不能在 defer 里 close， 否则不能保证所有数据都写入 buf。
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (c *Compressor) Uncompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = reader.Close()
	}()
	uncompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return uncompressed, nil
}
