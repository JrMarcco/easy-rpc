package easyrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"reflect"

	"github.com/JrMarcco/easy-rpc/message"
)

var _ Proxy = (*Server)(nil)

type Server struct {
	services map[string]*ProxyStub
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

func (s *Server) Register(service Service) {
	s.services[service.Name()] = &ProxyStub{
		service: service,
		refVal:  reflect.ValueOf(service),
	}
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
		return resp, fmt.Errorf("service %s not found", req.Service)
	}

	return ps.Call(ctx, req)
}

func NewServer() *Server {
	return &Server{
		services: make(map[string]*ProxyStub, 8),
	}
}

type ProxyStub struct {
	service Service
	refVal  reflect.Value
}

func (p *ProxyStub) Call(ctx context.Context, req *message.Req) (*message.Resp, error) {
	method := p.refVal.MethodByName(req.Method)

	inTyp := method.Type().In(1)
	in := reflect.New(inTyp.Elem())

	err := json.Unmarshal(req.Body, in.Interface())
	if err != nil {
		return nil, err
	}

	out := method.Call([]reflect.Value{reflect.ValueOf(ctx), in})
	respBody, err := json.Marshal(out[0].Interface())
	if err != nil {
		return nil, err
	}

	resp := &message.Resp{
		MessageId: req.MessageId,
		Body:      respBody,
	}

	if len(out) > 1 && !out[1].IsZero() {
		resp.Err = []byte(out[1].Interface().(error).Error())
	}

	resp.SetLength()
	return resp, nil
}
