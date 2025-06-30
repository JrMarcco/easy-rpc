package easyrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/JrMarcco/easy-rpc/message"
	"github.com/silenceper/pool"
)

var _ Proxy = (*Client)(nil)

type Client struct {
	connPool pool.Pool
}

func (c *Client) Call(_ context.Context, req *message.Req) (*message.Resp, error) {
	val, err := c.connPool.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer func() {
		_ = c.connPool.Put(val)
	}()

	conn := val.(net.Conn)

	_, err = conn.Write(message.EncodeReq(req))
	if err != nil {
		return nil, err
	}

	respBs, err := ReadMsg(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
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

				reqBody, err := json.Marshal(in)
				if err != nil {
					return []reflect.Value{reflect.ValueOf(out), reflect.ValueOf(err)}
				}

				req := &message.Req{
					Service: service.Name(),
					Method:  fd.Name,
					Body:    reqBody,
				}
				req.SetLength()

				// args[0] = context.Context
				resp, err := c.Call(args[0].Interface().(context.Context), req)
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
					err = json.Unmarshal(resp.Body, out)
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

func NewClient(addr string, initCap int, MaxCap int, MaxIdle int, idleTimeout time.Duration) (*Client, error) {
	connPool, err := pool.NewChannelPool(&pool.Config{
		InitialCap:  initCap,
		MaxCap:      MaxCap,
		MaxIdle:     MaxIdle,
		IdleTimeout: idleTimeout,
		Factory: func() (any, error) {
			return net.Dial("tcp", addr)
		},
		Close: func(i any) error {
			return i.(net.Conn).Close()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}
	return &Client{connPool: connPool}, nil
}
