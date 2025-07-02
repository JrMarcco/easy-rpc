package compress

var _ Compressor = (*DoNothing)(nil)

type DoNothing struct {
}

func (d *DoNothing) Code() uint8 {
	return CompressorNone
}

func (d *DoNothing) Compress(data []byte) ([]byte, error) {
	return data, nil
}

func (d *DoNothing) Uncompress(data []byte) ([]byte, error) {
	return data, nil
}
