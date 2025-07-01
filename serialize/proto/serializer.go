package proto

import (
	"fmt"

	"github.com/JrMarcco/easy-rpc/serialize"
	"google.golang.org/protobuf/proto"
)

var _ serialize.Serializer = (*Serializer)(nil)

type Serializer struct {
}

func (s *Serializer) Code() uint8 {
	return serialize.SerializerProto
}

func (s *Serializer) Marshal(val any) ([]byte, error) {
	if m, ok := val.(proto.Message); ok {
		return proto.Marshal(m)
	}
	return nil, fmt.Errorf("[easy-rpc] val must be proto.Message, but got: %T", val)
}

func (s *Serializer) Unmarshal(data []byte, val any) error {
	if m, ok := val.(proto.Message); ok {
		return proto.Unmarshal(data, m)
	}
	return fmt.Errorf("[easy-rpc] val must be proto.Message, but got: %T", val)
}
