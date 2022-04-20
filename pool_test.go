package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	Create(PoolConfig{})
}

func TestSimpleAcquire(t *testing.T) {
	assert := assert.New(t)

	pool := Create(PoolConfig{
		max: 1,
		create: func() any {
			return 1
		},
	})

	item, err := pool.Acquire()

	assert.Equal(1, item.Get())
	assert.Nil(err)
}

func TestExceddMaximumResources(t *testing.T) {
	pool := Create(PoolConfig{
		max: 0,
		create: func() any {
			return 1
		},
	})

	_, err := pool.Acquire()

	assert.NotNil(t, err)
}

func TestSimpleRelease(t *testing.T) {
	defer func() {
		assert.Nil(t, recover())
	}()

	pool := Create(PoolConfig{
		max: 1,
		create: func() any {
			return 1
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
		create: func() any {
			return 1
		},
	})

	pool.Acquire()
	pool.Release(Resource{
		id: 10,
	})
}

func TestMultipleResources(t *testing.T) {
	assert := assert.New(t)

	counter := 0
	pool := Create(PoolConfig{
		max: 2,
		create: func() any {
			counter++

			return counter
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
		create: func() any {
			counter++

			return counter
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
}

func TestDestroy(t *testing.T) {
	counter := 0
	isDestroyed := false

	pool := Create(PoolConfig{
		max: 2,
		create: func() any {
			counter++

			return counter
		},
		destroy: func(resource Resource) {
			isDestroyed = true
		},
	})

	resource, _ := pool.Acquire()
	pool.Release(resource)
	pool.Destroy()

	assert.True(t, isDestroyed)
}

func TestDestroyWhileThereAreLendedResource(t *testing.T) {
	defer func() {
		assert.NotNil(t, recover())
	}()

	counter := 0
	pool := Create(PoolConfig{
		max: 2,
		create: func() any {
			counter++

			return counter
		},
	})

	pool.Acquire()
	pool.Destroy()
}
