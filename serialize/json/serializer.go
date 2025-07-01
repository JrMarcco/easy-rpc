package json

import (
	sdkjson "encoding/json"

	"github.com/JrMarcco/easy-rpc/serialize"
)

var _ serialize.Serializer = (*Serializer)(nil)

type Serializer struct{}

func (s *Serializer) Code() uint8 {
	return serialize.SerializerJson
}

func (s *Serializer) Marshal(val any) ([]byte, error) {
	return sdkjson.Marshal(val)
}

func (s *Serializer) Unmarshal(data []byte, val any) error {
	return sdkjson.Unmarshal(data, val)
}
