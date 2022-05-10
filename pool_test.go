package gpool

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	Create(PoolConfig{
		Max: 10,
		Factory: PoolFactory{
			Create: func() any {
				return 1
			},
		},
	})
}

func TestCreateWithoutFactoryCreateMethod(t *testing.T) {
	assert.PanicsWithValue(t, "PoolConfig.Factory.Create is not provided", func() { Create(PoolConfig{}) })
}

func TestCreateWithoutMax(t *testing.T) {
	assert.PanicsWithValue(t, "PoolConfig.Max must be 1 or greater", func() {
		Create(PoolConfig{
			Factory: PoolFactory{
				Create: func() any {
					return 1
				},
			},
		})
	})
}

func TestAcquire(t *testing.T) {
	pool := Create(PoolConfig{
		Max: 1,
		Factory: PoolFactory{
			Create: func() any {
				return 1
			},
		},
	})

	item, err := pool.Acquire(nil)

	assert.Equal(t, 1, item.Get())
	assert.Nil(t, err)
}

func TestRelease(t *testing.T) {
	pool := Create(PoolConfig{
		Max: 1,
		Factory: PoolFactory{
			Create: func() any {
				return 1
			},
		},
	})

	resource, _ := pool.Acquire(nil)
	assert.NotPanics(t, func() { pool.Release(resource) })
}

func TestRelaseUnknownResource(t *testing.T) {
	pool := Create(PoolConfig{
		Max: 1,
		Factory: PoolFactory{
			Create: func() any {
				return 1
			},
		},
	})

	pool.Acquire(nil)
	assert.EqualError(
		t,
		pool.Release(Resource{
			id: "10",
		}),
		ErrorUnknownResource.Error(),
	)
}

func TestAcquireMultipleResources(t *testing.T) {
	assert := assert.New(t)

	counter := 0
	pool := Create(PoolConfig{
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
				counter++

				return counter
			},
		},
	})

	resource1, _ := pool.Acquire(nil)
	resource2, _ := pool.Acquire(nil)

	pool.Release(resource1)

	resource3, _ := pool.Acquire(nil)

	assert.Equal(1, resource1.Get())
	assert.Equal(2, resource2.Get())
	assert.Equal(1, resource3.Get())
	assert.Equal(resource1.id, resource3.id)
}

func TestPoolSize(t *testing.T) {
	assert := assert.New(t)

	counter := 0
	pool := Create(PoolConfig{
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
				counter++

				return counter
			},
		},
	})

	resource1, _ := pool.Acquire(nil)
	resource2, _ := pool.Acquire(nil)

	assert.Equal(2, pool.Size())
	assert.Equal(0, pool.NumberOfIdleResources())
	assert.Equal(2, pool.NumberOfLendedResources())

	pool.Release(resource1)

	assert.Equal(2, pool.Size())
	assert.Equal(1, pool.NumberOfIdleResources())
	assert.Equal(1, pool.NumberOfLendedResources())

	pool.Release(resource2)

	assert.Equal(2, pool.Size())
	assert.Equal(2, pool.NumberOfIdleResources())
	assert.Equal(0, pool.NumberOfLendedResources())

	pool.Acquire(nil)

	assert.Equal(2, pool.Size())
	assert.Equal(1, pool.NumberOfIdleResources())
	assert.Equal(1, pool.NumberOfLendedResources())
}

func TestDestroyPool(t *testing.T) {
	counter := 0
	isDestroyed := false

	pool := Create(PoolConfig{
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
				counter++

				return counter
			},
			Destroy: func(resource Resource) {
				isDestroyed = true
			},
		},
	})

	resource, _ := pool.Acquire(nil)
	pool.Release(resource)
	pool.DestroyPool()

	assert.True(t, isDestroyed)
}

func TestDestroyPoolWithUnreleasedResources(t *testing.T) {
	counter := 0

	pool := Create(PoolConfig{
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
				counter++

				return counter
			},
		},
	})

	pool.Acquire(nil)

	assert.EqualError(t, pool.DestroyPool(), ErrorCantDestroyPoolWithLendedResources.Error())
}

func TestInteractionAfterPoolIsDestroyed(t *testing.T) {
	counter := 0

	pool := Create(PoolConfig{
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
				counter++

				return counter
			},
		},
	})

	resource, _ := pool.Acquire(nil)
	pool.Release(resource)

	pool.DestroyPool()

	_, acquireError := pool.Acquire(nil)
	destroyError := pool.Destroy(Resource{})
	releaseError := pool.Release(Resource{})
	destroyPoolError := pool.DestroyPool()

	assert.EqualError(t, acquireError, ErrorPoolInactive.Error())
	assert.EqualError(t, destroyError, ErrorPoolInactive.Error())
	assert.EqualError(t, releaseError, ErrorPoolInactive.Error())
	assert.EqualError(t, destroyPoolError, ErrorPoolInactive.Error())
	assert.Equal(t, 0, pool.Size())
	assert.Equal(t, 0, pool.NumberOfLendedResources())
	assert.Equal(t, 0, pool.NumberOfIdleResources())
	assert.Equal(t, PoolStateInactive, pool.State())
}

func TestDestroyResource(t *testing.T) {
	counter := 0
	pool := Create(PoolConfig{
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
				counter++

				return counter
			},
		},
	})

	resource, _ := pool.Acquire(nil)
	pool.Destroy(resource)

	assert.Equal(t, 0, pool.Size())
}

func TestDestroyResourceWithDestroyFactoryMethod(t *testing.T) {
	counter := 0
	isDestroyed := false

	pool := Create(PoolConfig{
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
				counter++

				return counter
			},
			Destroy: func(resource Resource) {
				isDestroyed = true
			},
		},
	})

	resource, _ := pool.Acquire(nil)
	pool.Destroy(resource)

	assert.True(t, isDestroyed)
}

func TestDestroyAlreadyReleasedResource(t *testing.T) {
	counter := 0
	pool := Create(PoolConfig{
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
				counter++

				return counter
			},
		},
	})

	resource, _ := pool.Acquire(nil)
	pool.Release(resource)

	assert.EqualError(t, pool.Destroy(resource), ErrorUnknownResource.Error())
	assert.Equal(t, 1, pool.Size())
}

func TestDestroyAlreadyDestroyedResource(t *testing.T) {
	counter := 0
	pool := Create(PoolConfig{
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
				counter++

				return counter
			},
		},
	})

	resource, _ := pool.Acquire(nil)
	pool.Destroy(resource)

	assert.EqualError(t, pool.Destroy(resource), ErrorUnknownResource.Error())
	assert.Equal(t, 0, pool.Size())
}

// Should skip the first creatd resource, because it's invalid
func TestValidateResource(t *testing.T) {
	counter := 0
	pool := Create(PoolConfig{
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
				counter++

				return counter
			},
			Validate: func(resource Resource) bool {
				return resource.Get() != 1
			},
		},
	})

	resource, _ := pool.Acquire(nil)

	assert.Equal(t, 2, resource.Get())
	assert.Equal(t, 1, pool.Size())
}

func TestValidateInfinitely(t *testing.T) {
	counter := 0
	pool := Create(PoolConfig{
		Max:               2,
		AcquireTimeout:    time.Millisecond,
		MaxWaitingClients: 1,
		Factory: PoolFactory{
			Create: func() any {
				counter++

				return counter
			},
			Validate: func(resource Resource) bool {
				return false
			},
		},
	})

	pool.Acquire(nil)
	pool.Acquire(nil)
	_, err := pool.Acquire(nil)

	assert.EqualError(t, err, ErrorMaximumWaitingClientsExceeded.Error())
}

func TestConcureny(t *testing.T) {
	counter := 0
	pool := Create(PoolConfig{
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
				counter++

				return counter
			},
		},
	})

	for i := 0; i < 10; i++ {
		go func() {
			resource, err := pool.Acquire(nil)
			time.Sleep(time.Millisecond)
			if err == nil {
				pool.Release(resource)
			}
		}()
	}

	time.Sleep(time.Second * 2)

	assert.Equal(t, 2, pool.Size())
	assert.Equal(t, 2, pool.NumberOfIdleResources())
	assert.Equal(t, 0, pool.NumberOfLendedResources())
}

func TestAcquireTimeout(t *testing.T) {
	pool := Create(PoolConfig{
		Max:            2,
		AcquireTimeout: time.Millisecond * 10,
		Factory: PoolFactory{
			Create: func() any {
				return 1
			},
		},
	})

	pool.Acquire(nil)
	pool.Acquire(nil)
	_, err := pool.Acquire(nil)

	assert.EqualError(t, err, ErrorAcquireTimeout.Error())
}

func TestConcurentAcquireTimeout(t *testing.T) {
	pool := Create(PoolConfig{
		Max:            2,
		AcquireTimeout: time.Second,
		Factory: PoolFactory{
			Create: func() any {
				return 1
			},
		},
	})

	res, _ := pool.Acquire(nil)
	pool.Acquire(nil)

	go func(res Resource) {
		defer func() {
			recover()
			pool.Release(res)
		}()
		pool.Release(Resource{})
	}(res)

	_, err := pool.Acquire(nil)

	assert.Nil(t, err)
}

func TestSizeWithConcurentAcquireTimeout(t *testing.T) {
	pool := Create(PoolConfig{
		Max:            2,
		AcquireTimeout: time.Millisecond * 1,
		Factory: PoolFactory{
			Create: func() any {
				time.Sleep(time.Millisecond * 2)
				return 1
			},
		},
	})

	ch := make(chan int)
	for i := 0; i < 10; i++ {
		go func() {
			pool.Acquire(nil)
			ch <- 0
		}()
	}

	for i := 0; i < 10; i++ {
		<-ch
	}

	assert.Equal(t, pool.Size(), 2)
	assert.Equal(t, 0, pool.NumberOfIdleResources())
	assert.Equal(t, 2, pool.NumberOfLendedResources())
}

func TestMaxWaitingClients(t *testing.T) {
	pool := Create(PoolConfig{
		Max:               2,
		MaxWaitingClients: 1,
		Factory: PoolFactory{
			Create: func() any {
				return 1
			},
		},
	})

	pool.Acquire(nil)
	pool.Acquire(nil)

	go func() {
		pool.Acquire(nil)
	}()

	time.Sleep(time.Millisecond)

	_, err := pool.Acquire(nil)

	assert.EqualError(t, err, ErrorMaximumWaitingClientsExceeded.Error())
	assert.Equal(t, pool.Size(), 2)
	assert.Equal(t, 0, pool.NumberOfIdleResources())
	assert.Equal(t, 2, pool.NumberOfLendedResources())
}

func TestContextCancel(t *testing.T) {
	counter := 0
	pool := Create(PoolConfig{
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
				counter++

				return counter
			},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := pool.Acquire(ctx)

	assert.EqualError(t, err, ErrorAcquireTimeout.Error())
}

func TestContextCancelWithAcquireTimeout(t *testing.T) {
	counter := 0
	pool := Create(PoolConfig{
		Max:            2,
		AcquireTimeout: time.Minute,
		Factory: PoolFactory{
			Create: func() any {
				counter++

				return counter
			},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := pool.Acquire(ctx)

	assert.EqualError(t, err, ErrorAcquireTimeout.Error())
}
