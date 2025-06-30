package message

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResp(t *testing.T) {
	tcs := []struct {
		name string
		resp *Resp
	}{
		{
			name: "basic",
			resp: &Resp{
				HeadLen: 12,
				BodyLen: 12,
				ReqId:   1,

				Err:  []byte("test-err"),
				Body: []byte("test-data"),
			},
		}, {
			name: "without err",
			resp: &Resp{
				HeadLen: 12,
				BodyLen: 12,
				ReqId:   1,
				Body:    []byte("test-data"),
			},
		}, {
			name: "without body",
			resp: &Resp{
				HeadLen: 12,
				BodyLen: 12,
				ReqId:   1,
				Err:     []byte("test-err"),
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.resp.setHeadLen()
			tc.resp.setBodyLen()

			data := EncodeResp(tc.resp)
			decoded := DecodeResp(data)

			assert.Equal(t, tc.resp, decoded)
		})
	}
}
