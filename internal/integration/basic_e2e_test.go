//go:build e2e

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	easyrpc "github.com/JrMarcco/easy-rpc"
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
	SayHello func(ctx context.Context, req *testReq) (*testResp, error)
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
	return &testResp{
		Msg: fmt.Sprintf("hello %s", req.Name),
	}, nil
}

func TestBasicRemoteCall(t *testing.T) {
	svr := easyrpc.NewServer()
	ss := &testServerService{}
	svr.Register(ss)

	go func() {
		err := svr.Start(":8081")
		require.NoError(t, err)
	}()
	time.Sleep(time.Second)

	cs := &testClientService{}
	client, err := easyrpc.NewClient(":8081", 4, 16, 8, time.Second)
	require.NoError(t, err)

	client.InitService(cs)

	resp, err := cs.SayHello(context.Background(), &testReq{Name: "jrmarcco"})
	require.NoError(t, err)
	require.Equal(t, "hello jrmarcco", resp.Msg)
}
