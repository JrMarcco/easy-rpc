//go:build e2e

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	easyrpc "github.com/JrMarcco/easy-rpc"
	"github.com/JrMarcco/easy-rpc/compress/gzip"
	"github.com/JrMarcco/easy-rpc/internal/integration/pb"
	"github.com/JrMarcco/easy-rpc/serialize/proto"
	"github.com/stretchr/testify/require"
)

type testReq struct {
	Name string
}

type testResp struct {
	Msg string
}

var _ easyrpc.Service = (*testClientService)(nil)

type testClientService struct {
	SayHello      func(ctx context.Context, req *testReq) (*testResp, error)
	SayHelloProto func(ctx context.Context, req *pb.TestReq) (*pb.TestResp, error)
}

func (cs *testClientService) Name() string {
	return "test-service"
}

var _ easyrpc.Service = (*testServerService)(nil)

type testServerService struct {
}

func (ss *testServerService) Name() string {
	return "test-service"
}

func (ss *testServerService) SayHello(_ context.Context, req *testReq) (*testResp, error) {
	time.Sleep(time.Millisecond)
	return &testResp{
		Msg: fmt.Sprintf("hello %s", req.Name),
	}, nil
}

func (ss *testServerService) SayHelloProto(_ context.Context, req *pb.TestReq) (*pb.TestResp, error) {
	return &pb.TestResp{
		Msg: fmt.Sprintf("hello %s", req.Name),
	}, nil
}

func TestBasicRemoteCall(t *testing.T) {
	svr := easyrpc.NewServer()
	svr.RegisterService(&testServerService{})

	go func() {
		err := svr.Start(":8081")
		require.NoError(t, err)
	}()
	time.Sleep(100 * time.Millisecond)

	cs := &testClientService{}
	client, err := easyrpc.NewClientBuilder(":8081").Build()
	require.NoError(t, err)

	client.InitService(cs)

	resp, err := cs.SayHello(context.Background(), &testReq{Name: "jrmarcco"})
	require.NoError(t, err)
	require.Equal(t, "hello jrmarcco", resp.Msg)
}

func TestBasicRemoteCallProto(t *testing.T) {
	svr := easyrpc.NewServer()
	svr.RegisterService(&testServerService{})

	go func() {
		err := svr.Start(":8081")
		require.NoError(t, err)
	}()
	time.Sleep(100 * time.Millisecond)

	cs := &testClientService{}
	client, err := easyrpc.NewClientBuilder(":8081").
		Compressor(&gzip.Compressor{}).
		Serializer(&proto.Serializer{}).
		Build()
	require.NoError(t, err)

	client.InitService(cs)

	resp, err := cs.SayHelloProto(context.Background(), &pb.TestReq{
		Name: "jrmarcco",
	})
	require.NoError(t, err)
	require.Equal(t, "hello jrmarcco", resp.Msg)
}

func TestCompressRemoteCall(t *testing.T) {
	svr := easyrpc.NewServer()
	svr.RegisterService(&testServerService{})

	go func() {
		err := svr.Start(":8081")
		require.NoError(t, err)
	}()
	time.Sleep(100 * time.Millisecond)

	cs := &testClientService{}
	client, err := easyrpc.NewClientBuilder(":8081").
		Compressor(&gzip.Compressor{}).
		Build()
	require.NoError(t, err)

	client.InitService(cs)

	resp, err := cs.SayHello(context.Background(), &testReq{Name: "jrmarcco"})
	require.NoError(t, err)
	require.Equal(t, "hello jrmarcco", resp.Msg)
}

func TestTimeoutRemoteCall(t *testing.T) {
	svr := easyrpc.NewServer()
	svr.RegisterService(&testServerService{})

	go func() {
		err := svr.Start(":8081")
		require.NoError(t, err)
	}()
	time.Sleep(100 * time.Millisecond)

	cs := &testClientService{}
	client, err := easyrpc.NewClientBuilder(":8081").Build()
	require.NoError(t, err)

	client.InitService(cs)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond))
	resp, err := cs.SayHello(ctx, &testReq{Name: "jrmarcco"})
	cancel()

	require.Equal(t, context.DeadlineExceeded, err)
	require.NotNil(t, resp)
}
