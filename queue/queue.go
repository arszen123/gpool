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

type QueueIterator struct {
	queue     *Queue
	cursor    int
	isStarted bool
}

type NextValue struct {
	Done  bool
	Value any
}

// Iterator returns a new iterator that makes available to iterate over the items stored in the Queue.
func (q *Queue) Iterator() QueueIterator {
	return QueueIterator{
		queue: q,
	}
}

// Next returns the next item in the queue.
func (i *QueueIterator) Next() NextValue {
	i.advanceCursor()

	if i.isCursorOutOfRange() {
		return NextValue{
			Done:  true,
			Value: nil,
		}
	}

	return NextValue{
		Done:  false,
		Value: i.queue.items[i.cursor],
	}
}

// Remove removes the current item from the Queue.
func (i *QueueIterator) Remove() {
	if i.isCursorOutOfRange() {
		return
	}

	min := i.cursor
	max := i.cursor + 1

	items := i.queue.items
	items = append(items[:min], items[max:]...)

	i.decreaseCursor()

	i.queue.items = items
}

func (i QueueIterator) isCursorOutOfRange() bool {
	return i.cursor >= len(i.queue.items)
}

func (i *QueueIterator) advanceCursor() {
	if !i.isStarted {
		i.isStarted = true
		i.cursor = 0
		return
	}

	i.cursor++
}

func (i *QueueIterator) decreaseCursor() {
	i.cursor--

	if i.cursor < 0 {
		i.isStarted = false
		i.cursor = 0
		return
	}

}
