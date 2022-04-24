package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	Create(PoolConfig{
		max: 10,
		factory: PoolFactory{
			create: func() any {
				return 1
			},
		},
	})
}

func TestCreateWithoutFactoryCreateMethod(t *testing.T) {
	defer func() {
		assert.NotNil(t, recover())
	}()
	Create(PoolConfig{})
}

func TestSimpleAcquire(t *testing.T) {
	assert := assert.New(t)

	pool := Create(PoolConfig{
		max: 1,
		factory: PoolFactory{
			create: func() any {
				return 1
			},
		},
	})

	item, err := pool.Acquire()

	assert.Equal(1, item.Get())
	assert.Nil(err)
}

func TestMaximumResources(t *testing.T) {
	defer func() {
		assert.NotNil(t, recover())
	}()

	Create(PoolConfig{
		max: 0,
		factory: PoolFactory{
			create: func() any {
				return 1
			},
		},
	})
}

func TestSimpleRelease(t *testing.T) {
	defer func() {
		assert.Nil(t, recover())
	}()

	pool := Create(PoolConfig{
		max: 1,
		factory: PoolFactory{
			create: func() any {
				return 1
			},
		},
	})

	resource, _ := pool.Acquire()
	pool.Release(resource)
}

func TestRelaseUnknownResource(t *testing.T) {
	defer func() {
		assert.NotNil(t, recover())
	}()

	pool := Create(PoolConfig{
		max: 1,
		factory: PoolFactory{
			create: func() any {
				return 1
			},
		},
	})

	pool.Acquire()
	pool.Release(Resource{
		id: "10",
	})
}

func TestMultipleResources(t *testing.T) {
	assert := assert.New(t)

	counter := 0
	pool := Create(PoolConfig{
		max: 2,

		factory: PoolFactory{
			create: func() any {
				counter++

				return counter
			},
		},
	})

	resource1, _ := pool.Acquire()
	resource2, _ := pool.Acquire()

	pool.Release(resource1)

	resource3, _ := pool.Acquire()

	assert.Equal(1, resource1.Get())
	assert.Equal(2, resource2.Get())
	assert.Equal(1, resource3.Get())
	assert.Equal(resource1.id, resource3.id)
}

func TestPoolSize(t *testing.T) {
	assert := assert.New(t)

	counter := 0
	pool := Create(PoolConfig{
		max: 2,
		factory: PoolFactory{
			create: func() any {
				counter++

				return counter
			},
		},
	})

	resource1, _ := pool.Acquire()
	resource2, _ := pool.Acquire()

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

	pool.Acquire()

	assert.Equal(2, pool.Size())
	assert.Equal(1, pool.NumberOfIdleResources())
	assert.Equal(1, pool.NumberOfLendedResources())
}

func TestDestroy(t *testing.T) {
	counter := 0
	isDestroyed := false

	pool := Create(PoolConfig{
		max: 2,
		factory: PoolFactory{
			create: func() any {
				counter++

				return counter
			},
			destroy: func(resource Resource) {
				isDestroyed = true
			},
		},
	})

	resource, _ := pool.Acquire()
	pool.Release(resource)
	pool.DestroyAll()

	assert.True(t, isDestroyed)
}

func TestDestroyWhileThereAreLendedResource(t *testing.T) {
	defer func() {
		assert.NotNil(t, recover())
	}()

	counter := 0
	pool := Create(PoolConfig{
		max: 2,
		factory: PoolFactory{
			create: func() any {
				counter++

				return counter
			},
		},
	})

	pool.Acquire()
	pool.DestroyAll()
}

func TestDestroyResource(t *testing.T) {
	counter := 0
	pool := Create(PoolConfig{
		max: 2,
		factory: PoolFactory{
			create: func() any {
				counter++

				return counter
			},
		},
	})

	resource, _ := pool.Acquire()
	pool.Destroy(resource)

	assert.Equal(t, 0, pool.Size())
}

// Should skip the first creatd resource, because it's invalid
func TestValidateResource(t *testing.T) {
	counter := 0
	pool := Create(PoolConfig{
		max: 2,
		factory: PoolFactory{
			create: func() any {
				counter++

				return counter
			},
			validate: func(resource Resource) bool {
				return resource.Get() != 1
			},
		},
	})

	resource, _ := pool.Acquire()

	assert.Equal(t, 2, resource.Get())
	assert.Equal(t, 1, pool.Size())
}

func TestConcureny(t *testing.T) {
	counter := 0
	pool := Create(PoolConfig{
		max: 2,
		factory: PoolFactory{
			create: func() any {
				counter++

				return counter
			},
		},
	})

	for i := 0; i < 10; i++ {
		go func() {
			resource, err := pool.Acquire()
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
