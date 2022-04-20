package queue

import "testing"

func TestCreate(t *testing.T) {
	queue := Create()

	if queue.Size() != 0 {
		t.Fatalf("Queue actual size is %d, but should be %d", queue.Size(), 3)
	}
}

func TestEnqueu(t *testing.T) {
	item := 1
	queue := Queue{}
	queue.Enqueue(item)

	if queue.Size() != 1 {
		t.Fatal("Failed to enqueue item")
	}
}

func TestDequeue(t *testing.T) {
	queue := Queue{}
	queue.Enqueue(1)
	queue.Enqueue(2)
	queue.Enqueue(3)
	receivedItem := queue.Dequeue()

	if receivedItem != 1 {
		t.Fatalf("Wrong item is requeued. expected: %d, received: %d", 1, receivedItem)
	}

	if queue.Size() != 2 {
		t.Fatal("Item still in the queue")
	}
}

func TestSize(t *testing.T) {
	queue := Queue{}
	queue.Enqueue(1)
	queue.Enqueue(2)
	queue.Enqueue(3)

	if queue.Size() != 3 {
		t.Fatalf("Queue actual size is %d, but should be %d", queue.Size(), 3)
	}
}

func TestDequeueEmpty(t *testing.T) {
	queue := Queue{}
	receivedItem := queue.Dequeue()

	if receivedItem != nil {
		t.Fatalf("Wrong item is returned. expected: nil, received: %d", receivedItem)
	}
}
