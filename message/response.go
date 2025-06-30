package message

import (
	"encoding/binary"
)

// Resp rpc 响应信息
//
// | 	  head length 4  	| 	body length 4 	|
// |      message id  4     |
// |  	  error message	    |
// | 	  response body	    |
type Resp struct {
	HeadLen uint32
	BodyLen uint32

	MessageId uint32

	Err  []byte
	Body []byte
}

func (resp *Resp) SetLength() {
	// 设置 head 长度
	resp.HeadLen = 12 + uint32(len(resp.Err))
	// 设置 body 长度
	resp.BodyLen = uint32(len(resp.Body))
}

func EncodeResp(resp *Resp) []byte {
	bs := make([]byte, resp.HeadLen+resp.BodyLen)

	// 写入 head 长度
	binary.BigEndian.PutUint32(bs[:4], resp.HeadLen)
	// 写入 body 长度
	binary.BigEndian.PutUint32(bs[4:8], resp.BodyLen)
	// 写入 message id
	binary.BigEndian.PutUint32(bs[8:12], resp.MessageId)

	// 写入 err
	copy(bs[12:resp.HeadLen], resp.Err)
	// 写入 body
	copy(bs[resp.HeadLen:], resp.Body)

	return bs
}

func DecodeResp(data []byte) *Resp {
	resp := &Resp{}

	// 解码 head 长度
	resp.HeadLen = binary.BigEndian.Uint32(data[:4])
	// 解码 body 长度
	resp.BodyLen = binary.BigEndian.Uint32(data[4:8])
	// 解码 message id
	resp.MessageId = binary.BigEndian.Uint32(data[8:12])

	// 解码 err
	if resp.HeadLen > 12 {
		resp.Err = data[12:resp.HeadLen]
	}
	// 解码 body
	if resp.BodyLen > 0 {
		resp.Body = data[resp.HeadLen : resp.HeadLen+resp.BodyLen]
	}
	return resp
}
