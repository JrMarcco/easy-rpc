package message

import (
	"bytes"
	"encoding/binary"
)

const (
	separator = '\n'
	equalSign = '\t'
)

// Req rpc 请求信息
//
// | 	  head length 4  	| 					   	 body length 4           		      |
// |      message id 	    |  version 1 | compressor 1 |  serializer 1 | message type 1  |
// | 	  service name 		|
// | 	  method name 		|
// | 	  meta ...			|
// | 	  meta ...			|
// | 	  	...				|
// |	  request  body		|
//
// service/method/meta 之间使用分隔符 ':' 隔开
type Req struct {
	HeadLen uint32
	BodyLen uint32

	ReqId       uint32
	Version     uint8
	Compressor  uint8
	Serializer  uint8
	MessageType uint8

	Service string
	Method  string
	Body    []byte

	Meta map[string]string
}

func (req *Req) setHeadLen() {
	// +2 是因为 service/method 之间共有 2 个分隔符
	headLen := 16 + len(req.Service) + len(req.Method) + 2

	if len(req.Meta) > 0 {
		for k, v := range req.Meta {
			// + 2 是因为每条条 meta 都有一个等号和一个分隔符
			headLen += len(k) + len(v) + 2
		}
	}

	req.HeadLen = uint32(headLen)
}

func (req *Req) setBodyLen() {
	req.BodyLen = uint32(len(req.Body))
}

// EncodeReq 将 rpc 请求编码成二进制
func EncodeReq(req *Req) []byte {
	bs := make([]byte, req.HeadLen+req.BodyLen)

	// 写入 head 长度
	binary.BigEndian.PutUint32(bs[:4], req.HeadLen)
	// 写入 body 长度
	binary.BigEndian.PutUint32(bs[4:8], req.BodyLen)

	// 写入 request id
	binary.BigEndian.PutUint32(bs[8:12], req.ReqId)
	// 写入 version
	bs[12] = req.Version
	// 写入 compressor
	bs[13] = req.Compressor
	// 写入 serializer
	bs[14] = req.Serializer
	// 写入 message type
	bs[15] = req.MessageType

	// 写入 service
	curr := bs[16:]
	copy(curr, req.Service)
	// 写入分隔符
	curr = curr[len(req.Service):]
	curr[0] = separator
	// 写入 method
	curr = curr[1:]
	copy(curr, req.Method)

	// 写入 meta
	curr = curr[len(req.Method):]
	curr[0] = separator
	curr = curr[1:]

	for k, v := range req.Meta {
		copy(curr, k)
		curr = curr[len(k):]
		curr[0] = equalSign
		curr = curr[1:]
		copy(curr, v)
		curr = curr[len(v):]
		curr[0] = separator
		curr = curr[1:]
	}

	// 写入 body
	copy(curr, req.Body)

	return bs
}

// DecodeReq 将二进制信息解码为 rpc 请求信息
func DecodeReq(data []byte) *Req {
	req := &Req{}

	// 解码 head 长度
	req.HeadLen = binary.BigEndian.Uint32(data[:4])
	// 解码 body 长度
	req.BodyLen = binary.BigEndian.Uint32(data[4:8])

	// 解码 request id
	req.ReqId = binary.BigEndian.Uint32(data[8:12])
	// 解码 version
	req.Version = data[12]
	// 解码 compressor
	req.Compressor = data[13]
	// 解码 serializer
	req.Serializer = data[14]
	// 解码 message type
	req.MessageType = data[15]

	// 解码 service
	head := data[16:req.HeadLen]
	index := bytes.IndexByte(head, separator)
	req.Service = string(head[:index])

	// 解码 method
	head = head[index+1:]
	index = bytes.IndexByte(head, separator)
	req.Method = string(head[:index])

	// 解码 meta
	head = head[index+1:]
	index = bytes.IndexByte(head, separator)
	if index != -1 {
		meta := make(map[string]string, 4)
		for index != -1 {
			md := head[:index]
			mdIndex := bytes.IndexByte(md, equalSign)

			meta[string(md[:mdIndex])] = string(md[mdIndex+1:])

			head = head[index+1:]
			index = bytes.IndexByte(head, separator)
		}

		req.Meta = meta
	}

	// 解码 body
	if req.BodyLen != 0 {
		req.Body = data[req.HeadLen:]
	}
	return req
}
