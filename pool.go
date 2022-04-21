package pool

import (
	"errors"

	"github.com/arszen123/gpool/queue"
	"github.com/google/uuid"
)

type Pool struct {
	resources       queue.Queue
	allResources    []Resource
	lendedResources []Resource
	config          PoolConfig
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
	state    ResourceState
}
type ResourceState string

const (
	RESOURCE_STATE_IDLE      ResourceState = "IDLE"
	RESOURCE_STATE_ALLOCATED               = "ALLOCATED"
	RESOURCE_STATE_INVALID                 = "INVALID"
)

func Create(config PoolConfig) Pool {
	assertFactoryMethodsAvailable(config)

	return Pool{
		resources: queue.Create(),
		config:    config,
	}
}

func (p *Pool) Acquire() (Resource, error) {
	resource := p.resources.Dequeue()

	if resource == nil {
		if p.Size() >= p.config.max {
			return Resource{}, errors.New("Maximum acquired items reached")
		}

		item := p.config.factory.create()
		resource = Resource{
			id:       uuid.NewString(),
			resource: item,
			state:    RESOURCE_STATE_ALLOCATED,
		}
		p.allResources = append(p.allResources, resource.(Resource))
	}

	p.lendedResources = append(p.lendedResources, resource.(Resource))

	if p.config.factory.validate != nil && !p.config.factory.validate(resource.(Resource)) {
		p.Destroy(resource.(Resource))
		return p.Acquire()
	}

	return resource.(Resource), nil
}

func (p *Pool) Release(resource Resource) {
	idx, err := p.getLendedResourceIndex(resource)

	if err != nil {
		panic("Can't release an unknown resource")
	}

	resource.state = RESOURCE_STATE_IDLE
	p.lendedResources = append(p.lendedResources[:idx], p.lendedResources[idx+1:]...)
	p.resources.Enqueue(resource)
}

func (p *Pool) Destroy(resource Resource) {
	idx, err := p.getLendedResourceIndex(resource)

	if err != nil {
		panic("Can't destroy an unknown resource")
	}

	p.lendedResources = append(p.lendedResources[:idx], p.lendedResources[idx+1:]...)
	p.destroy(resource)
}

func (p *Pool) destroy(resource Resource) {
	resource.state = RESOURCE_STATE_INVALID
	p.removeResource(resource)
	if p.config.factory.destroy != nil {
		p.config.factory.destroy(resource)
	}
}

func (p *Pool) removeResource(resource Resource) {
	for idx, aResource := range p.allResources {
		if aResource.id == resource.id {
			p.allResources = append(p.allResources[:idx], p.allResources[idx+1:]...)
			break
		}
	}
}

func (p *Pool) DestroyAll() {
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

func (p Pool) getLendedResourceIndex(item Resource) (int, error) {
	for idx, lendedItem := range p.lendedResources {
		if lendedItem.id == item.id {
			return idx, nil
		}
	}

	return 0, errors.New("Resource not found")
}

func assertFactoryMethodsAvailable(config PoolConfig) {
	factory := config.factory

	if factory.create == nil {
		panic("factory.create is not provided")
	}
}

func (r Resource) Get() any {
	return r.resource
}
