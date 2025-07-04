package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/JrMarcco/easy-rpc/registry"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var eventTypeMap = map[mvccpb.Event_EventType]registry.EventType{
	mvccpb.PUT:    registry.EventTypePut,
	mvccpb.DELETE: registry.EventTypeDel,
}

var _ registry.Registry = (*Registry)(nil)

type Registry struct {
	mu          sync.Mutex
	etcdClient  *clientv3.Client
	etcdSession *concurrency.Session

	watchCancel []context.CancelFunc
}

func (r *Registry) Register(ctx context.Context, instance registry.ServiceInstance) error {
	val, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	_, err = r.etcdClient.Put(ctx, r.instanceKey(instance), string(val), clientv3.WithLease(r.etcdSession.Lease()))
	return err
}

func (r *Registry) Unregister(ctx context.Context, instance registry.ServiceInstance) error {
	_, err := r.etcdClient.Delete(ctx, r.instanceKey(instance))
	return err
}

func (r *Registry) ListServices(ctx context.Context, serviceName string) ([]registry.ServiceInstance, error) {
	resp, err := r.etcdClient.Get(ctx, r.serviceKey(serviceName), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	res := make([]registry.ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var instance registry.ServiceInstance
		err = json.Unmarshal(kv.Value, &instance)
		if err != nil {
			return nil, err
		}
		res = append(res, instance)
	}
	return res, nil
}

func (r *Registry) Subscribe(serviceName string) <-chan registry.Event {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = clientv3.WithRequireLeader(ctx)

	r.mu.Lock()
	r.watchCancel = append(r.watchCancel, cancel)
	r.mu.Unlock()

	watchChan := r.etcdClient.Watch(ctx, r.serviceKey(serviceName), clientv3.WithPrefix())

	ch := make(chan registry.Event)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case resp := <-watchChan:
				if resp.Err() != nil {
					continue
				}
				if resp.Canceled {
					return
				}

				for _, e := range resp.Events {
					ch <- registry.Event{Type: eventTypeMap[e.Type]}
				}
			}
		}
	}()

	return ch
}

func (r *Registry) Close() error {
	r.mu.Lock()
	for _, cancel := range r.watchCancel {
		cancel()
	}
	r.mu.Unlock()

	return r.etcdSession.Close()
}

func (r *Registry) serviceKey(serviceName string) string {
	return fmt.Sprintf("/easyrpc/%s", serviceName)
}

func (r *Registry) instanceKey(instance registry.ServiceInstance) string {
	return fmt.Sprintf("/easyrpc/%s/%s", instance.Name, instance.Addr)
}

func NewRegistry(client *clientv3.Client) (*Registry, error) {
	session, err := concurrency.NewSession(client)
	if err != nil {
		return nil, err
	}
	return &Registry{
		etcdClient:  client,
		etcdSession: session,
	}, nil
}
