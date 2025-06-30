package message

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReq(t *testing.T) {
	tcs := []struct {
		name string
		req  *Req
	}{
		{
			name: "basic",
			req: &Req{
				MessageId: 1,

				Version:     1,
				Serializer:  1,
				Compressor:  1,
				MessageType: 1,

				Service: "test-service",
				Method:  "test-method",
				Body:    []byte("test-data"),

				Meta: map[string]string{
					"test-key-1": "test-value-1",
					"test-key-2": "test-value-2",
					"test-key-3": "test-value-3",
				},
			},
		}, {
			name: "without body",
			req: &Req{
				MessageId: 1,

				Version:     1,
				Serializer:  1,
				Compressor:  1,
				MessageType: 1,

				Service: "test-service",
				Method:  "test-method",

				Meta: map[string]string{
					"test-key-1": "test-value-1",
					"test-key-2": "test-value-2",
					"test-key-3": "test-value-3",
				},
			},
		}, {
			name: "without meta",
			req: &Req{
				MessageId: 1,

				Version:     1,
				Serializer:  1,
				Compressor:  1,
				MessageType: 1,

				Service: "test-service",
				Method:  "test-method",
				Body:    []byte("test-data"),
			},
		}, {
			name: "body with separator",
			req: &Req{
				MessageId: 1,

				Version:     1,
				Serializer:  1,
				Compressor:  1,
				MessageType: 1,

				Service: "test-service",
				Method:  "test-method",
				Body:    []byte(fmt.Sprintf("test-data%ctest-data", separator)),

				Meta: map[string]string{
					"test-key-1": "test-value-1",
					"test-key-2": "test-value-2",
					"test-key-3": "test-value-3",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.req.SetLength()

			data := EncodeReq(tc.req)
			decoded := DecodeReq(data)

			assert.Equal(t, tc.req, decoded)
		})
	}
}
