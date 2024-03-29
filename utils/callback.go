package utils

import (
	"context"
	"fmt"
	"log"
	"sync"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

// Callbacks 类似于http架构的中间件作用
type Callbacks struct {
	Signal         chan struct{}
	Debug          bool
	Fetches        int
	Requests       int
	DeltaRequests  int
	DeltaResponses int
	mu             sync.Mutex
}

var _ server.Callbacks = &Callbacks{Debug: true}

func (cb *Callbacks) Report() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	log.Printf("server callbacks fetches=%d requests=%d\n", cb.Fetches, cb.Requests)
}
func (cb *Callbacks) OnStreamOpen(_ context.Context, id int64, typ string) error {
	if cb.Debug {
		log.Printf("stream %d open for %s\n", id, typ)
	}
	return nil
}
func (cb *Callbacks) OnStreamClosed(id int64, node *core.Node) {
	if cb.Debug {
		log.Printf("stream %d of node %s closed\n", id, node.Id)
	}
}
func (cb *Callbacks) OnDeltaStreamOpen(_ context.Context, id int64, typ string) error {
	if cb.Debug {
		log.Printf("delta stream %d open for %s\n", id, typ)
	}
	return nil
}
func (cb *Callbacks) OnDeltaStreamClosed(id int64, node *core.Node) {
	if cb.Debug {
		log.Printf("delta stream %d of node %s closed\n", id, node.Id)
	}
}
func (cb *Callbacks) OnStreamRequest(int64, *discovery.DiscoveryRequest) error {
	fmt.Println(1111)
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.Requests++
	if cb.Signal != nil {
		close(cb.Signal)
		cb.Signal = nil
	}
	return nil
}
func (cb *Callbacks) OnStreamResponse(context.Context, int64, *discovery.DiscoveryRequest, *discovery.DiscoveryResponse) {
	fmt.Println(2222)
}
func (cb *Callbacks) OnStreamDeltaResponse(id int64, req *discovery.DeltaDiscoveryRequest, res *discovery.DeltaDiscoveryResponse) {
	fmt.Println(3)
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.DeltaResponses++
}
func (cb *Callbacks) OnStreamDeltaRequest(id int64, req *discovery.DeltaDiscoveryRequest) error {
	fmt.Println(4)
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.DeltaRequests++
	if cb.Signal != nil {
		close(cb.Signal)
		cb.Signal = nil
	}

	return nil
}
func (cb *Callbacks) OnFetchRequest(_ context.Context, req *discovery.DiscoveryRequest) error {
	fmt.Println(5)
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.Fetches++
	if cb.Signal != nil {
		close(cb.Signal)
		cb.Signal = nil
	}
	return nil
}
func (cb *Callbacks) OnFetchResponse(*discovery.DiscoveryRequest, *discovery.DiscoveryResponse) {
	fmt.Println(6)

}
