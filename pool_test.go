package pool

import (
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
	defer func() {
		assert.NotNil(t, recover())
	}()
	Create(PoolConfig{})
}

func TestSimpleAcquire(t *testing.T) {
	assert := assert.New(t)

	pool := Create(PoolConfig{
		Max: 1,
		Factory: PoolFactory{
			Create: func() any {
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
		Max: 0,
		Factory: PoolFactory{
			Create: func() any {
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
		Max: 1,
		Factory: PoolFactory{
			Create: func() any {
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
		Max: 1,
		Factory: PoolFactory{
			Create: func() any {
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
		Max: 2,

		Factory: PoolFactory{
			Create: func() any {
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
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
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
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
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
		Max: 2,
		Factory: PoolFactory{
			Create: func() any {
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

	resource, _ := pool.Acquire()

	assert.Equal(t, 2, resource.Get())
	assert.Equal(t, 1, pool.Size())
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

func TestAcquireTimeout(t *testing.T) {
	pool := Create(PoolConfig{
		Max:                  2,
		AcquireTimeoutMillis: time.Millisecond * 10,
		Factory: PoolFactory{
			Create: func() any {
				return 1
			},
		},
	})

	pool.Acquire()
	pool.Acquire()
	_, err := pool.Acquire()

	assert.EqualError(t, err, "Acquire timeout")
}

func TestAcquireTimeoutWithUnknownReleas(t *testing.T) {
	pool := Create(PoolConfig{
		Max:                  2,
		AcquireTimeoutMillis: time.Second,
		Factory: PoolFactory{
			Create: func() any {
				return 1
			},
		},
	})

	res, _ := pool.Acquire()
	pool.Acquire()
	go func(res Resource) {
		defer func() {
			recover()
			pool.Release(res)
		}()
		pool.Release(Resource{})
	}(res)
	_, err := pool.Acquire()

	assert.Nil(t, err)
}

func TestSizeWithAcquireTimeout(t *testing.T) {
	pool := Create(PoolConfig{
		Max:                  2,
		AcquireTimeoutMillis: time.Millisecond * 1,
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
			_, err := pool.Acquire()
			if err == nil {
				ch <- 1
			} else {
				ch <- 0
			}
		}()
	}

	sum := 0
	for i := 0; i < 5; i++ {
		v := <-ch
		sum += v
	}

	assert.Equal(t, pool.Size()-sum, pool.NumberOfIdleResources())
	assert.Equal(t, sum, pool.NumberOfLendedResources())
}
