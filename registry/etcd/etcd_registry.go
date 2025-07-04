package etcd

import (
	"context"
	"encoding/json"
	"errors"
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

	leaseTTL int // 租约 ttl

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

type RegistryBuilder struct {
	etcdClient *clientv3.Client

	// 租约 ttl，默认 30s
	//
	// | cluster scale | recommended lease TTL (seconds) |
	// |    < 50       |            30                   |
	// |    50 ~ 200   |            15                   |
	// |    > 200      |            10                   |
	//
	// 集群规模越大续约越频繁的原因是：
	// 1、服务变更感知时效性要求更高。
	// 2、减少“僵尸”服务的影响。
	// 3、防止单点故障影响扩大。
	// 4、更快的健康检查与剔除。
	// 5、分摊注册中心压力。
	//		虽然续约频繁会增加 etcd 的 QPS，但大集群本身对注册中心的压力主要来自服务变更和查询。
	//		但是通过合理缩短 TTL 和加快续约，可以让 etcd 更及时地维护服务列表，避免因“过期”服务过多导致的查询不准确。
	leaseTTL int
}

// LeaseTTL 设置租约 ttl，单位为秒。
func (rb *RegistryBuilder) LeaseTTL(ttl int) *RegistryBuilder {
	rb.leaseTTL = ttl
	return rb
}

func (rb *RegistryBuilder) Build() (*Registry, error) {
	if rb.leaseTTL <= 0 {
		return nil, errors.New("[jotify] etcd lease TTL must be greater than 0")
	}

	session, err := concurrency.NewSession(rb.etcdClient, concurrency.WithTTL(rb.leaseTTL))
	if err != nil {
		return nil, err
	}
	return &Registry{
		etcdClient:  rb.etcdClient,
		etcdSession: session,
	}, nil
}

func NewRegistryBuilder(client *clientv3.Client) *RegistryBuilder {
	return &RegistryBuilder{
		etcdClient: client,
		leaseTTL:   30,
	}
}
