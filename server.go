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
		resp, err := s.Call(context.Background(), req)
		if err != nil {
			resp.Err = []byte(err.Error())
		}

		_, err = conn.Write(message.EncodeResp(resp))
		if err != nil {
			return
		}
	}
}

func (s *Server) Call(ctx context.Context, req *message.Req) (*message.Resp, error) {
	resp := &message.Resp{
		MessageId: req.MessageId,
	}

	ps, ok := s.services[req.Service]
	if !ok {
		return resp, fmt.Errorf("[easy-rpc] service %s not found", req.Service)
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
	resp := &message.Resp{
		MessageId: req.MessageId,
	}
	defer resp.SetLength()

	// 获取 serializer
	serializer, ok := p.serializers[req.Serializer]
	if !ok {
		return resp, fmt.Errorf("[easy-rpc] unsupported serializer of code %c", req.Serializer)
	}

	// 获取调用方法
	method := p.refVal.MethodByName(req.Method)

	inTyp := method.Type().In(1)
	in := reflect.New(inTyp.Elem())

	err := serializer.Unmarshal(req.Body, in.Interface())
	if err != nil {
		return resp, err
	}

	// 实际方法调用
	out := method.Call([]reflect.Value{reflect.ValueOf(ctx), in})
	respBody, err := serializer.Marshal(out[0].Interface())
	if err != nil {
		return resp, err
	}
	resp.Body = respBody

	if len(out) > 1 && !out[1].IsZero() {
		resp.Err = []byte(out[1].Interface().(error).Error())
	}
	return resp, nil
}
