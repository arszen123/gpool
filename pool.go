// Simple resource pool.
package gpool

import (
	"errors"
	"sync"
	"time"

	"github.com/arszen123/gpool/queue"
	"github.com/google/uuid"
)

var (
	ErrorMaximumWaitingClientsExceeded = errors.New("Maximum waiting clients exceeded")
	ErrorNoResourceAvailable           = errors.New("No resource available")
	ErrorAcquireTimeout                = errors.New("Acquire timeout")
)

type Pool struct {
	resources       queue.Queue
	allResources    []Resource
	lendedResources []Resource
	config          PoolConfig
	mux             sync.Mutex
	waitingClients  []chan Resource
}

type PoolFactory struct {
	Create   func() any
	Destroy  func(resource Resource)
	Validate func(resource Resource) bool
}

type PoolConfig struct {
	Max                  int
	AcquireTimeoutMillis time.Duration
	MaxWaitingClients    int
	Factory              PoolFactory
}

// Resource holds the client resource.
type Resource struct {
	id       string
	resource any
}

// Create returns a new Pool.
func Create(config PoolConfig) Pool {
	assertPoolConfig(config)

	return Pool{
		resources:      queue.Create(),
		config:         config,
		mux:            sync.Mutex{},
		waitingClients: []chan Resource{},
	}
}

// Acquire retrieves a Resource from the pool.
func (p *Pool) Acquire() (Resource, error) {
	ch := make(chan Resource)

	p.mux.Lock()
	maxWaitingClients := p.config.MaxWaitingClients
	if maxWaitingClients > 0 && len(p.waitingClients) > maxWaitingClients {
		return Resource{}, ErrorMaximumWaitingClientsExceeded
	}
	p.waitingClients = append(p.waitingClients, ch)
	p.mux.Unlock()

	go p.dispatch()

	if p.config.AcquireTimeoutMillis <= 0 {
		resource, ok := <-ch
		if !ok {
			return Resource{}, ErrorNoResourceAvailable
		}
		return resource, nil
	}

	select {
	case resource, ok := <-ch:
		if !ok {
			return Resource{}, ErrorNoResourceAvailable
		}
		return resource, nil
	case <-time.After(p.config.AcquireTimeoutMillis):
		go func() {
			resource, ok := <-ch
			if ok {
				p.Release(resource)
			}
		}()
		return Resource{}, ErrorAcquireTimeout
	}
}

func (p *Pool) dispatch() {
	p.mux.Lock()
	if len(p.waitingClients) <= 0 || (p.Size() >= p.config.Max && p.resources.Size() <= 0) {
		p.mux.Unlock()
		return
	}

	ch := p.waitingClients[0]
	p.waitingClients = p.waitingClients[1:]
	resource := p.resources.Dequeue()

	if resource == nil {
		item := p.config.Factory.Create()
		resource = Resource{
			id:       uuid.NewString(),
			resource: item,
		}
		p.allResources = append(p.allResources, resource.(Resource))
	}

	p.lendedResources = append(p.lendedResources, resource.(Resource))
	p.mux.Unlock()

	if p.config.Factory.Validate != nil && !p.config.Factory.Validate(resource.(Resource)) {
		var err error

		p.Destroy(resource.(Resource))
		resource, err = p.Acquire()
		if err != nil {
			close(ch)
			return
		}
	}
	ch <- resource.(Resource)
}

// Release releases a resources.
func (p *Pool) Release(resource Resource) {
	p.mux.Lock()
	defer func() {
		go p.dispatch()
	}()
	defer p.mux.Unlock()

	idx, err := getResourceIndex(p.lendedResources, resource)

	if err != nil {
		panic("Can't release an unknown resource")
	}

	p.lendedResources = append(p.lendedResources[:idx], p.lendedResources[idx+1:]...)
	p.resources.Enqueue(resource)
}

// Destroy destroys a resource.
func (p *Pool) Destroy(resource Resource) {
	p.mux.Lock()
	defer func() {
		go p.dispatch()
	}()
	defer p.mux.Unlock()

	idx, err := getResourceIndex(p.lendedResources, resource)

	if err != nil {
		panic("Can't destroy an unknown resource")
	}

	p.lendedResources = append(p.lendedResources[:idx], p.lendedResources[idx+1:]...)
	p.destroy(resource)
}

func (p *Pool) destroy(resource Resource) {
	idx, err := getResourceIndex(p.allResources, resource)
	if err != nil {
		return
	}

	p.allResources = append(p.allResources[:idx], p.allResources[idx+1:]...)

	if p.config.Factory.Destroy != nil {
		p.config.Factory.Destroy(resource)
	}
}

// DestroyAll desytroys all the resources in the pool.
func (p *Pool) DestroyAll() {
	p.mux.Lock()
	defer p.mux.Unlock()
	if p.config.Factory.Destroy == nil {
		panic("PoolConfig.fatory.destroy is not provided")
	}
	if len(p.lendedResources) > 0 {
		panic("Can't destroy pool when there are lended resources")
	}

	resource := p.resources.Dequeue()
	for resource != nil {

		p.config.Factory.Destroy(resource.(Resource))

		resource = p.resources.Dequeue()
	}
}

// Size returns the number of resources the pool manages.
func (p Pool) Size() int {
	return len(p.allResources)
}

// NumberOfLendedResources returns the number of lended resources.
func (p Pool) NumberOfLendedResources() int {
	return len(p.lendedResources)
}

// NumberOfIdleResources returns the number of available resources.
func (p Pool) NumberOfIdleResources() int {
	return p.resources.Size()
}

func getResourceIndex(items []Resource, item Resource) (int, error) {
	for idx, lendedItem := range items {
		if lendedItem.id == item.id {
			return idx, nil
		}
	}

	return 0, errors.New("Resource not found")
}

func assertPoolConfig(config PoolConfig) {
	factory := config.Factory

	if factory.Create == nil {
		panic("PoolConfig.factory.create is not provided")
	}
	if config.Max <= 0 {
		panic("PoolConfig.max must be 1 or greate")
	}
}

// Get returns the resource.
func (r Resource) Get() any {
	return r.resource
}
