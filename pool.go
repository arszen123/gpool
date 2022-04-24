package pool

import (
	"errors"
	"sync"

	"github.com/arszen123/gpool/queue"
	"github.com/google/uuid"
)

type Pool struct {
	resources       queue.Queue
	allResources    []Resource
	lendedResources []Resource
	config          PoolConfig
	mux             sync.Mutex
}
type PoolFactory struct {
	create   func() any
	destroy  func(resource Resource)
	validate func(resource Resource) bool
}

type PoolConfig struct {
	max     int
	factory PoolFactory
}

type Resource struct {
	id       string
	resource any
}

func Create(config PoolConfig) Pool {
	assertPoolConfig(config)

	return Pool{
		resources: queue.Create(),
		config:    config,
		mux:       sync.Mutex{},
	}
}

func (p *Pool) Acquire() (Resource, error) {
	p.mux.Lock()

	resource := p.resources.Dequeue()

	if resource == nil {
		if p.Size() >= p.config.max {
			p.mux.Unlock()
			return Resource{}, errors.New("Maximum acquired items reached")
		}

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
		p.Destroy(resource.(Resource))
		return p.Acquire()
	}

	return resource.(Resource), nil
}

func (p *Pool) Release(resource Resource) {
	p.mux.Lock()
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
		panic("fatory.destroy is not provided")
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
