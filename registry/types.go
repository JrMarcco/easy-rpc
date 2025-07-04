package registry

import (
	"context"
	"io"
)

type Registry interface {
	Register(ctx context.Context, instance ServiceInstance) error
	Unregister(ctx context.Context, instance ServiceInstance) error
	ListServices(ctx context.Context, serviceName string) ([]ServiceInstance, error)
	Subscribe(serviceName string) <-chan Event
	io.Closer
}

type ServiceInstance struct {
	Name  string
	Addr  string
	Group string
}

type EventType uint8

//goland:noinspection GoUnusedConst
const (
	EventTypeUnknow EventType = iota
	EventTypePut
	EventTypeDel
)

type Event struct {
	Type            EventType
	ServiceInstance ServiceInstance
}
