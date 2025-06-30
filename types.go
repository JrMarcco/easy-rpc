package easyrpc

import (
	"context"

	"github.com/JrMarcco/easy-rpc/message"
)

//go:generate mockgen -source=./types.go -destination=./mock/proxy.mock.go -package=proxymock -typed Proxy

type Service interface {
	Name() string
}

type Proxy interface {
	Call(ctx context.Context, req *message.Req) (*message.Resp, error)
}
