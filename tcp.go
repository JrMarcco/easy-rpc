package easyrpc

import (
	"encoding/binary"
	"fmt"
	"net"
)

const lenBytes = 8

func ReadMsg(conn net.Conn) (bs []byte, err error) {
	lenBs := make([]byte, lenBytes)
	_, err = conn.Read(lenBs)
	if err != nil {
		return nil, fmt.Errorf("[easy-rpc] failed to read length: %w", err)
	}

	// 读取长度字段
	headLen := binary.BigEndian.Uint32(lenBs[:4])
	bodyLen := binary.BigEndian.Uint32(lenBs[4:])
	length := headLen + bodyLen

	bs = make([]byte, length)
	_, err = conn.Read(bs[8:])
	if err != nil {
		return nil, fmt.Errorf("[easy-rpc] failed to read message: %w", err)
	}
	copy(bs[:8], lenBs)
	return bs, nil
}
