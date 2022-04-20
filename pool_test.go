package pool

import (
	"testing"
)

func TestCreate(t *testing.T) {
	Create(PoolConfig{})
}

func TestSimpleAcquire(t *testing.T) {
	pool := Create(PoolConfig{
		max: 1,
		create: func() any {
			return 1
		},
	})

	item, err := pool.Acquire()

	if item.Get() != 1 {
		t.Fatalf("Expected item %d, received %d", 1, item)
	}

	if err != nil {
		t.Fatal("Acquire should not return an error and a resource")
	}
}

func TestExceddMaximumResources(t *testing.T) {
	pool := Create(PoolConfig{
		max: 0,
		create: func() any {
			return 1
		},
	})

	_, err := pool.Acquire()

	if err == nil {
		t.Fatal("Acquire should fail because reached the maximum number of allowed resources")
	}
}

func TestSimpleRelease(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatal("Should not fail")
		}
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
		if r := recover(); r == nil {
			t.Fatal("Should fail")
		}
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

	if 1 != resource1.Get() {
		t.Fatalf("1. resource expected to be %d, received %d", 1, resource1.Get())
	}

	if 2 != resource2.Get() {
		t.Fatalf("2. resource expected to be %d, received %d", 2, resource2.Get())
	}

	if 1 != resource3.Get() {
		t.Fatalf("3. resource expected to be %d, received %d", 1, resource3.Get())
	}

	if resource1.id != resource3.id {
		t.Fatalf("3. resource expected to have id %d, received %d", resource1.id, resource3.id)
	}
}

func TestPoolSize(t *testing.T) {
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

	if pool.Size() != 2 {
		t.Fatalf("Pool size expected to be %d, received %d", 2, pool.Size())
	}

	if pool.NumberOfIdleResources() != 0 {
		t.Fatalf("Number of idle resources expected to be %d, received %d", 0, pool.NumberOfIdleResources())
	}

	if pool.NumberOfLendedResources() != 2 {
		t.Fatalf("Number of lended resources expected to be %d, received %d", 2, pool.NumberOfLendedResources())
	}

	pool.Release(resource1)

	if pool.Size() != 2 {
		t.Fatalf("Pool size expected to be %d, received %d", 2, pool.Size())
	}

	if pool.NumberOfIdleResources() != 1 {
		t.Fatalf("Number of idle resources expected to be %d, received %d", 1, pool.NumberOfIdleResources())
	}

	if pool.NumberOfLendedResources() != 1 {
		t.Fatalf("Number of lended resources expected to be %d, received %d", 1, pool.NumberOfLendedResources())
	}

	pool.Release(resource2)

	if pool.Size() != 2 {
		t.Fatalf("Pool size expected to be %d, received %d", 2, pool.Size())
	}

	if pool.NumberOfIdleResources() != 2 {
		t.Fatalf("Number of idle resources expected to be %d, received %d", 2, pool.NumberOfIdleResources())
	}

	if pool.NumberOfLendedResources() != 0 {
		t.Fatalf("Number of lended resources expected to be %d, received %d", 0, pool.NumberOfLendedResources())
	}
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

	if !isDestroyed {
		t.Fatal("Should have destroyed the resources")
	}
}

func TestDestroyWhileThereAreLendedResource(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Should fail")
		}
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
