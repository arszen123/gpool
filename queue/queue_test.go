package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	queue := Create()

	assert.Equal(t, 0, queue.Size())
}

func TestEnqueu(t *testing.T) {
	item := 1
	queue := Queue{}
	queue.Enqueue(item)

	assert.Equal(t, 1, queue.Size())
}

func TestDequeue(t *testing.T) {
	queue := Queue{}
	queue.Enqueue(1)
	queue.Enqueue(2)
	queue.Enqueue(3)
	receivedItem := queue.Dequeue()

	assert.Equal(t, 1, receivedItem)
	assert.Equal(t, 2, queue.Size())
}

func TestSize(t *testing.T) {
	queue := Queue{}
	queue.Enqueue(1)
	queue.Enqueue(2)
	queue.Enqueue(3)

	assert.Equal(t, 3, queue.Size())
}

func TestDequeueEmpty(t *testing.T) {
	queue := Queue{}
	receivedItem := queue.Dequeue()

	assert.Nil(t, receivedItem, queue.Size())
}
