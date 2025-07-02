package easyrpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/JrMarcco/easy-rpc/compress"
	"github.com/JrMarcco/easy-rpc/message"
	"github.com/JrMarcco/easy-rpc/serialize"
	"github.com/JrMarcco/easy-rpc/serialize/json"
	"github.com/silenceper/pool"
)

var _ Proxy = (*Client)(nil)

type Client struct {
	connPool   pool.Pool
	compressor compress.Compressor
	serializer serialize.Serializer
}

func (c *Client) Call(ctx context.Context, req *message.Req) (resp *message.Resp, err error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	ch := make(chan struct{})
	defer close(ch)

	go func() {
		resp, err = c.sendRequest(ctx, req)
		ch <- struct{}{}
	}()

	select {
	// 监听超时
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ch:
		return resp, err
	}
}

func (c *Client) sendRequest(ctx context.Context, req *message.Req) (*message.Resp, error) {
	val, err := c.connPool.Get()
	if err != nil {
		return nil, fmt.Errorf("[easy-rpc] failed to get connection: %w", err)
	}
	defer func() {
		_ = c.connPool.Put(val)
	}()

	conn := val.(net.Conn)

	_, err = conn.Write(message.EncodeReq(req))
	if err != nil {
		return nil, err
	}

	// 如果是 oneway 调用，这里可以直接返回
	if isOneway(ctx) {
		return &message.Resp{
			MessageId: req.MessageId,
		}, nil
	}

	respBs, err := ReadMsg(conn)
	if err != nil {
		return nil, fmt.Errorf("[easy-rpc] failed to read response: %w", err)
	}
	return message.DecodeResp(respBs), nil
}

func (c *Client) InitService(service Service) {
	c.setProxyFunc(service)
}

func (c *Client) setProxyFunc(service Service) {
	val := reflect.ValueOf(service)
	elem := val.Elem()
	typ := elem.Type()

	numField := typ.NumField()
	for i := 0; i < numField; i++ {
		fdVal := elem.Field(i)
		if fdVal.CanSet() {
			fd := typ.Field(i)

			fn := func(args []reflect.Value) []reflect.Value {
				// req
				in := args[1].Interface()
				// resp
				out := reflect.New(fd.Type.Out(0).Elem()).Interface()

				reqBody, err := c.serializer.Marshal(in)
				if err != nil {
					return []reflect.Value{reflect.ValueOf(out), reflect.ValueOf(err)}
				}

				// 压缩 request body
				compressedBody, err := c.compressor.Compress(reqBody)
				if err != nil {
					return []reflect.Value{reflect.ValueOf(out), reflect.ValueOf(err)}
				}

				// args[0] = context.Context
				ctx := args[0].Interface().(context.Context)

				req := &message.Req{
					Compressor: c.compressor.Code(),
					Serializer: c.serializer.Code(),
					Service:    service.Name(),
					Method:     fd.Name,
					Body:       compressedBody,
					Meta:       c.metaFromContext(ctx),
				}
				req.SetLength()

				resp, err := c.Call(ctx, req)
				if err != nil {
					return []reflect.Value{reflect.ValueOf(out), reflect.ValueOf(err)}
				}

				// 处理 Resp.Err（服务端回传的错误）
				refErrVal := reflect.Zero(reflect.TypeOf(new(error)).Elem())
				if len(resp.Err) != 0 {
					remoteErr := errors.New(string(resp.Err))
					if remoteErr != nil {
						refErrVal = reflect.ValueOf(remoteErr)
					}
				}

				if resp.BodyLen > 0 {
					err = c.serializer.Unmarshal(resp.Body, out)
					if err != nil {
						return []reflect.Value{reflect.ValueOf(out), reflect.ValueOf(err)}
					}
				}

				return []reflect.Value{reflect.ValueOf(out), refErrVal}
			}
			fdVal.Set(reflect.MakeFunc(fd.Type, fn))
		}
	}
}

// metaFromContext 通过 context 构建 meta 数据。
func (c *Client) metaFromContext(ctx context.Context) map[string]string {
	meta := make(map[string]string, 2)
	if dl, ok := ctx.Deadline(); ok {
		// 设置了超时时间
		meta[metaKeyDeadline] = strconv.FormatInt(dl.UnixMilli(), 10)
	}
	if isOneway(ctx) {
		meta[metaKeyOneway] = "true"
	}
	return meta
}

type ClientBuilder struct {
	addr       string
	connPool   pool.Pool
	compressor compress.Compressor
	serializer serialize.Serializer
}

func (cb *ClientBuilder) ConnPool(pool pool.Pool) *ClientBuilder {
	cb.connPool = pool
	return cb
}

func (cb *ClientBuilder) Compressor(compressor compress.Compressor) *ClientBuilder {
	cb.compressor = compressor
	return cb
}

func (cb *ClientBuilder) Serializer(serializer serialize.Serializer) *ClientBuilder {
	cb.serializer = serializer
	return cb
}

func (cb *ClientBuilder) Build() (*Client, error) {
	if cb.connPool == nil {
		connPool, err := pool.NewChannelPool(&pool.Config{
			InitialCap:  8,
			MaxCap:      64,
			MaxIdle:     16,
			IdleTimeout: time.Minute,
			Factory:     func() (any, error) { return net.Dial("tcp", cb.addr) },
			Close:       func(val any) error { return val.(net.Conn).Close() },
		})
		if err != nil {
			return nil, fmt.Errorf("[easy-rpc] failed to create connection pool: %w", err)
		}
		cb.connPool = connPool
	}

	return &Client{
		connPool:   cb.connPool,
		compressor: cb.compressor,
		serializer: cb.serializer,
	}, nil
}

func NewClientBuilder(addr string) *ClientBuilder {
	return &ClientBuilder{
		addr:       addr,
		compressor: &compress.DoNothing{},
		serializer: &json.Serializer{},
	}
}
