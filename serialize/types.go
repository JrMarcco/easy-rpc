package serialize

const (
	SerializerJson  = 1
	SerializerProto = 2
)

type Serializer interface {
	Code() uint8
	Marshal(val any) ([]byte, error)
	Unmarshal(data []byte, val any) error
}
