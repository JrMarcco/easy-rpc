package json

import (
	"encoding/json"

	"github.com/JrMarcco/easy-rpc/serialize"
)

var _ serialize.Serializer = (*Serializer)(nil)

type Serializer struct{}

func (s *Serializer) Code() uint8 {
	return serialize.SerializerJson
}

func (s *Serializer) Marshal(val any) ([]byte, error) {
	return json.Marshal(val)
}

func (s *Serializer) Unmarshal(data []byte, val any) error {
	return json.Unmarshal(data, val)
}
