package easyrpc

import (
	"context"
	"fmt"
	"net"
	"reflect"

	"github.com/JrMarcco/easy-rpc/message"
	"github.com/JrMarcco/easy-rpc/serialize"
	"github.com/JrMarcco/easy-rpc/serialize/json"
	"github.com/JrMarcco/easy-rpc/serialize/proto"
)

var _ Proxy = (*Server)(nil)

type Server struct {
	services    map[string]*ProxyStub
	serializers map[uint8]serialize.Serializer
}

func (s *Server) Start(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}

		go s.handleConn(conn)
	}
}

func (s *Server) RegisterService(service Service) {
	s.services[service.Name()] = &ProxyStub{
		service:     service,
		refVal:      reflect.ValueOf(service),
		serializers: s.serializers,
	}
}

func (s *Server) RegisterSerializer(serializer serialize.Serializer) {
	s.serializers[serializer.Code()] = serializer
}

func (s *Server) handleConn(conn net.Conn) {
	for {
		reqBs, err := ReadMsg(conn)
		if err != nil {
			return
		}

		req := message.DecodeReq(reqBs)
		ctx := s.buildContext(req.Meta)

		resp, err := s.Call(ctx, req)
		if req.Meta[metaKeyOneway] == "true" {
			continue
		}
		if err != nil {
			resp = &message.Resp{
				MessageId: req.MessageId,
				Err:       []byte(err.Error()),
			}
		}

		resp.SetLength()
		_, err = conn.Write(message.EncodeResp(resp))
		if err != nil {
			return
		}
	}
}

func (s *Server) buildContext(meta map[string]string) context.Context {
	ctx := context.Background()
	if meta[metaKeyOneway] == "true" {
		ctx = ContextWithOneway(ctx)
	}
	return ctx
}

func (s *Server) Call(ctx context.Context, req *message.Req) (*message.Resp, error) {
	ps, ok := s.services[req.Service]
	if !ok {
		return nil, fmt.Errorf("[easy-rpc] service %s not found", req.Service)
	}

	if isOneway(ctx) {
		go func() {
			_, _ = ps.Call(ctx, req)
		}()
		return nil, nil
	}

	return ps.Call(ctx, req)
}

func NewServer() *Server {
	svr := &Server{
		services:    make(map[string]*ProxyStub, 8),
		serializers: make(map[uint8]serialize.Serializer, 2),
	}
	svr.RegisterSerializer(&json.Serializer{})
	svr.RegisterSerializer(&proto.Serializer{})
	return svr
}

type ProxyStub struct {
	service Service
	refVal  reflect.Value

	serializers map[uint8]serialize.Serializer
}

func (p *ProxyStub) Call(ctx context.Context, req *message.Req) (*message.Resp, error) {
	// 获取 serializer
	serializer, ok := p.serializers[req.Serializer]
	if !ok {
		return nil, fmt.Errorf("[easy-rpc] unsupported serializer of code %c", req.Serializer)
	}

	// 获取调用方法
	method := p.refVal.MethodByName(req.Method)

	inTyp := method.Type().In(1)
	in := reflect.New(inTyp.Elem())

	err := serializer.Unmarshal(req.Body, in.Interface())
	if err != nil {
		return nil, err
	}

	// 实际方法调用
	out := method.Call([]reflect.Value{reflect.ValueOf(ctx), in})
	if len(out) > 1 && !out[1].IsZero() {
		return nil, out[1].Interface().(error)
	}

	respBody, err := serializer.Marshal(out[0].Interface())
	if err != nil {
		return nil, err
	}

	return &message.Resp{
		MessageId: req.MessageId,
		Body:      respBody,
	}, nil
}
