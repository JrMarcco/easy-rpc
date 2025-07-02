package compress

const (
	CompressorNone = 0
	CompressorGzip = 1
)

type Compressor interface {
	Code() uint8
	Compress(data []byte) ([]byte, error)
	Uncompress(data []byte) ([]byte, error)
}
