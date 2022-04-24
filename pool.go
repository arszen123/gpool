package pool

import (
	"errors"
	"sync"
	"time"

	"github.com/arszen123/gpool/queue"
	"github.com/google/uuid"
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
	create   func() any
	destroy  func(resource Resource)
	validate func(resource Resource) bool
}

type PoolConfig struct {
	max                  int
	acquireTimeoutMillis time.Duration
	factory              PoolFactory
}

type Resource struct {
	id       string
	resource any
}

func Create(config PoolConfig) Pool {
	assertPoolConfig(config)

	return Pool{
		resources:      queue.Create(),
		config:         config,
		mux:            sync.Mutex{},
		waitingClients: []chan Resource{},
	}
}

func (p *Pool) Acquire() (Resource, error) {
	ch := make(chan Resource)

	p.mux.Lock()
	p.waitingClients = append(p.waitingClients, ch)
	p.mux.Unlock()

	go p.dispatch()

	if p.config.acquireTimeoutMillis <= 0 {
		resource, ok := <-ch
		if !ok {
			return Resource{}, errors.New("No resource available")
		}
		return resource, nil
	}

	select {
	case resource, ok := <-ch:
		if !ok {
			return Resource{}, errors.New("No resource available")
		}
		return resource, nil
	case <-time.After(p.config.acquireTimeoutMillis):
		go func() {
			resource, ok := <-ch
			if ok {
				p.Release(resource)
			}
		}()
		return Resource{}, errors.New("Acquire timeout")
	}
}

func (p *Pool) dispatch() {
	p.mux.Lock()
	if len(p.waitingClients) <= 0 || (p.Size() >= p.config.max && p.resources.Size() <= 0) {
		p.mux.Unlock()
		return
	}

	ch := p.waitingClients[0]
	p.waitingClients = p.waitingClients[1:]
	resource := p.resources.Dequeue()

	if resource == nil {
		item := p.config.factory.create()
		resource = Resource{
			id:       uuid.NewString(),
			resource: item,
		}
		p.allResources = append(p.allResources, resource.(Resource))
	}

	p.lendedResources = append(p.lendedResources, resource.(Resource))
	p.mux.Unlock()

	if p.config.factory.validate != nil && !p.config.factory.validate(resource.(Resource)) {
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

func (p *Pool) Release(resource Resource) {
	p.mux.Lock()
	defer func() {
		go p.dispatch()
	}()
	defer p.mux.Unlock()

	idx := getResourceIndex(p.lendedResources, resource)

	if idx < 0 {
		panic("Can't release an unknown resource")
	}

	p.lendedResources = append(p.lendedResources[:idx], p.lendedResources[idx+1:]...)
	p.resources.Enqueue(resource)
}

func (p *Pool) Destroy(resource Resource) {
	p.mux.Lock()
	defer func() {
		go p.dispatch()
	}()
	defer p.mux.Unlock()

	idx := getResourceIndex(p.lendedResources, resource)

	if idx < 0 {
		panic("Can't destroy an unknown resource")
	}

	p.lendedResources = append(p.lendedResources[:idx], p.lendedResources[idx+1:]...)
	p.destroy(resource)
}

func (p *Pool) destroy(resource Resource) {
	idx := getResourceIndex(p.allResources, resource)
	if idx >= 0 {
		p.allResources = append(p.allResources[:idx], p.allResources[idx+1:]...)
		if p.config.factory.destroy != nil {
			p.config.factory.destroy(resource)
		}
	}
}

func (p *Pool) DestroyAll() {
	p.mux.Lock()
	defer p.mux.Unlock()
	if p.config.factory.destroy == nil {
		panic("PoolConfig.fatory.destroy is not provided")
	}
	if len(p.lendedResources) > 0 {
		panic("Can't destroy pool when there are lended resources")
	}

	resource := p.resources.Dequeue()
	for resource != nil {

		p.config.factory.destroy(resource.(Resource))

		resource = p.resources.Dequeue()
	}
}

func (p Pool) Size() int {
	return len(p.allResources)
}

func (p Pool) NumberOfLendedResources() int {
	return len(p.lendedResources)
}

func (p Pool) NumberOfIdleResources() int {
	return p.resources.Size()
}

func getResourceIndex(items []Resource, item Resource) int {
	for idx, lendedItem := range items {
		if lendedItem.id == item.id {
			return idx
		}
	}

	return -1
}

func assertPoolConfig(config PoolConfig) {
	factory := config.factory

	if factory.create == nil {
		panic("PoolConfig.factory.create is not provided")
	}
	if config.max <= 0 {
		panic("PoolConfig.max must be 1 or greate")
	}
}

func (r Resource) Get() any {
	return r.resource
}
