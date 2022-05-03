package queue

type Queue struct {
	items []any
}

// Create creates a new empty queue.
func Create() Queue {
	return Queue{
		items: []any{},
	}
}

// Enqueue adds an item to the queue.
func (q *Queue) Enqueue(item any) {
	q.items = append(q.items, item)
}

// Dequeue removes an item from the queue.
func (q *Queue) Dequeue() any {
	if len(q.items) == 0 {
		return nil
	}

	item := q.items[0]
	q.items = q.items[1:]

	return item
}

// Size returns the number of items in the queue.
func (q Queue) Size() int {
	return len(q.items)
}
