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
	ErrorMaximumWaitingClientsExceeded      = errors.New("Maximum waiting clients exceeded")
	ErrorNoResourceAvailable                = errors.New("No resource available")
	ErrorAcquireTimeout                     = errors.New("Acquire timeout")
	ErrorUnknownResource                    = errors.New("Unknown resource")
	ErrorCantDestroyPoolWithLendedResources = errors.New("Can't destroy pool when there are lended resources")
	ErrorPoolInactive                       = errors.New("Can't perform actions on an inactive pool")
)

type PoolState = string

const (
	PoolStateActive   PoolState = "ACTIVE"
	PoolStateInactive           = "INACTIVE"
)

type Pool struct {
	resources       queue.Queue
	allResources    []Resource
	lendedResources []Resource
	config          PoolConfig
	mux             sync.Mutex
	waitingClients  []chan Resource
	state           PoolState
}

type PoolFactory struct {
	Create   func() any
	Destroy  func(resource Resource)
	Validate func(resource Resource) bool
}

type PoolConfig struct {
	Max               int
	AcquireTimeout    time.Duration
	MaxWaitingClients int
	Factory           PoolFactory
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
		state:          PoolStateActive,
	}
}

// Acquire retrieves a Resource from the pool.
func (p *Pool) Acquire() (Resource, error) {
	p.mux.Lock()

	if isPoolInactive(*p) {
		defer p.mux.Unlock()
		return Resource{}, ErrorPoolInactive
	}

	ch := make(chan Resource)
	maxWaitingClients := p.config.MaxWaitingClients

	if maxWaitingClients > 0 && len(p.waitingClients) >= maxWaitingClients {
		defer p.mux.Unlock()
		return Resource{}, ErrorMaximumWaitingClientsExceeded
	}

	p.waitingClients = append(p.waitingClients, ch)
	p.mux.Unlock()

	go p.dispatch()

	return p.resolveResource(ch)
}

func (p *Pool) resolveResource(ch chan Resource) (Resource, error) {
	createResponse := func(resource Resource, ok bool) (Resource, error) {
		if !ok {
			return Resource{}, ErrorNoResourceAvailable
		}
		return resource, nil
	}

	if p.config.AcquireTimeout <= 0 {
		resource, ok := <-ch
		return createResponse(resource, ok)
	}

	select {
	case resource, ok := <-ch:
		return createResponse(resource, ok)
	case <-time.After(p.config.AcquireTimeout):
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

	isPoolInactive := isPoolInactive(*p)

	if isPoolInactive || len(p.waitingClients) <= 0 || (p.Size() >= p.config.Max && p.resources.Size() <= 0) {
		if isPoolInactive && len(p.waitingClients) > 0 {
			for _, ch := range p.waitingClients {
				close(ch)
			}
			p.waitingClients = []chan Resource{}
		}
		p.mux.Unlock()
		return
	}

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
		p.Destroy(resource.(Resource))
		go p.dispatch()
		return
	}

	p.mux.Lock()

	ch := p.waitingClients[0]
	p.waitingClients = p.waitingClients[1:]

	p.mux.Unlock()

	ch <- resource.(Resource)
}

// Release releases a resources.
func (p *Pool) Release(resource Resource) error {
	p.mux.Lock()
	defer func() {
		go p.dispatch()
	}()
	defer p.mux.Unlock()

	if isPoolInactive(*p) {
		return ErrorPoolInactive
	}

	idx, err := getResourceIndex(p.lendedResources, resource)

	if err != nil {
		return ErrorUnknownResource
	}

	p.lendedResources = append(p.lendedResources[:idx], p.lendedResources[idx+1:]...)
	p.resources.Enqueue(resource)

	return nil
}

// Destroy destroys a resource.
func (p *Pool) Destroy(resource Resource) error {
	p.mux.Lock()
	defer func() {
		go p.dispatch()
	}()
	defer p.mux.Unlock()

	if isPoolInactive(*p) {
		return ErrorPoolInactive
	}

	idx, err := getResourceIndex(p.lendedResources, resource)

	if err != nil {
		return ErrorUnknownResource
	}

	p.lendedResources = append(p.lendedResources[:idx], p.lendedResources[idx+1:]...)
	p.destroy(resource)

	return nil
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

// DestroyPool desytroys all the resources in the pool.
func (p *Pool) DestroyPool() error {
	p.mux.Lock()
	defer p.mux.Unlock()

	if isPoolInactive(*p) {
		return ErrorPoolInactive
	}

	if len(p.lendedResources) > 0 {
		return ErrorCantDestroyPoolWithLendedResources
	}

	for _, resource := range p.allResources {

		if p.config.Factory.Destroy != nil {
			p.config.Factory.Destroy(resource)
		}
	}
	p.allResources = []Resource{}

	resource := p.resources.Dequeue()
	for resource != nil {
		resource = p.resources.Dequeue()
	}

	p.state = PoolStateInactive

	return nil
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

// State returns the pool state.
func (p Pool) State() PoolState {
	return p.state
}

func isPoolInactive(p Pool) bool {
	return p.state == PoolStateInactive
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
		panic("PoolConfig.Factory.Create is not provided")
	}
	if config.Max <= 0 {
		panic("PoolConfig.Max must be 1 or greater")
	}
}

// Get returns the resource.
func (r Resource) Get() any {
	return r.resource
}
