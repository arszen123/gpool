package pool

import (
	"errors"

	"github.com/arszen123/gpool/queue"
)

type Pool struct {
	resources       queue.Queue
	allResources    []Resource
	lendedResources []Resource
	config          PoolConfig
}

type PoolConfig struct {
	max     int
	create  func() any
	destroy func(resource Resource)
}

type Resource struct {
	id       int
	resource any
}

func Create(config PoolConfig) Pool {
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

		item := p.config.create()
		newResource := Resource{
			id:       p.Size() + 1,
			resource: item,
		}
		resource = newResource
		p.allResources = append(p.allResources, newResource)
		p.lendedResources = append(p.lendedResources, newResource)
	}

	return resource.(Resource), nil
}

func (p *Pool) Release(resource Resource) {
	idx, err := p.getLendedResourceIndex(resource)

	if err != nil {
		panic("Can't release an unknown resource")
	}

	p.lendedResources = append(p.lendedResources[:idx], p.lendedResources[idx+1:]...)
	p.resources.Enqueue(resource)
}

func (p *Pool) Destroy() {
	if len(p.lendedResources) > 0 {
		panic("Can't destroy pool when there are lended resources")
	}

	resource := p.resources.Dequeue()
	for resource != nil {

		p.config.destroy(resource.(Resource))

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

func (r Resource) Get() any {
	return r.resource
}
