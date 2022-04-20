package queue

type Queue struct {
	items []any
}

func Create() Queue {
	return Queue{
		items: []any{},
	}
}

func (q *Queue) Enqueue(item any) {
	q.items = append(q.items, item)
}

func (q *Queue) Dequeue() any {
	if len(q.items) == 0 {
		return nil
	}

	item := q.items[0]
	q.items = q.items[1:]

	return item
}

func (q Queue) Size() int {
	return len(q.items)
}
