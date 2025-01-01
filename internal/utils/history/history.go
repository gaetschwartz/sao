package history

const HistorySize = 2

type History[T any] interface {
	Add(T)
	Restore() (T, bool)
	Get() T
	Size() int
	SetSize(int)
}

type BoundedHistory[T any] struct {
	queue []T
	size  int
	// cursor is the index of the next element to be read
	// it grows to the right infinitely and we mod it with the size
	// to get the actual index.
	cursor  int
	written int
	History[T]
}

func NewBoundedHistory[T any](size int) *BoundedHistory[T] {
	if size < 1 {
		size = 1
	}
	return &BoundedHistory[T]{
		queue:   make([]T, size),
		size:    size,
		cursor:  0,
		written: 0,
	}
}

// Push an element to the queue
func (q *BoundedHistory[T]) Add(elem T) {
	if q.written == 0 {
		q.cursor = 0
	} else {
		q.cursor += 1
	}
	q.queue[q.cursor%q.size] = elem
	q.written += 1
}

func (q *BoundedHistory[T]) AddAll(elems []T) {
	// only take the last `q.size` elements
	last := elems[min(0, len(elems)-q.size):]
	cur := q.cursor % q.size
	space_after_cursor := q.size - cur
	// copy the elements that fit after the cursor
	copy(
		q.queue[cur:],
		last[:space_after_cursor],
	)
	q.written += space_after_cursor
	// if there are more elements than space after the cursor, we need to copy them
	if len(last) > space_after_cursor {
		// copy the elements that don't fit after the cursor
		copy(q.queue[:len(last)-space_after_cursor], last[space_after_cursor:])
		q.written += len(last) - space_after_cursor
	}
	q.cursor = q.written - 1
}

// Pop an element from the queue
func (q *BoundedHistory[T]) Restore() (T, bool) {
	new_cursor := q.cursor - 1
	// check if the new cursor would be at a legal position
	// i.e not out of bounds
	lower_bound := q.written - q.size
	// fmt.Println("Trying to restore: ", q.cursor, "=>", new_cursor, "with lower bound", lower_bound, "written", q.written)
	if new_cursor >= lower_bound {
		// we can restore
		// fmt.Println("can restore at", new_cursor)
		q.cursor = new_cursor
		return q.queue[q.cursor%q.size], true
	} else {
		// fmt.Println("can't restore at", new_cursor)
		// we can't restore, so we return current element
		return q.queue[q.cursor%q.size], false
	}
}

// Peek at the next element in the queue
func (q *BoundedHistory[T]) Get() T {
	return q.queue[q.cursor%q.size]
}

func (q *BoundedHistory[T]) Size() int {
	return q.size
}

// Resize is a bit tricky, we need to copy the values that are in the queue
// before the cursor, and after the cursor.
// We have two main cases:
//  1. New size is bigger than the current size
//  2. New size is smaller than the current size
//
// # 1. new_size > q.size
//
// # 1.1 cursor > size
//
// |            cursor
// |              v
// | [1, 2, 3, 4, 5, 6, 7, 8]
// | [ <- newer -> ][ older ]
// |                       v new_cursor
// | [1, 2, 3, 4, 5, 6, 7, 8, x, x]
// | [ older ][ <- newer -> ]
//
// # 1.2 cursor < size
//
// |            cursor
// |              v
// | [1, 2, 3, 4, 5, 6, 7, 8]
// | [ <- newer -> ][ trash ]
// |              v new_cursor
// | [1, 2, 3, 4, 5, 6, 7, 8, x, x]
// | [ <- newer -> ]
//
// # 2. new_size < q.size
//
// # 2.1 cursor > size
//
// |            cursor
// |              v
// | [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
// | [ <- newer -> ][ <- older  -> ]
// |                              v new_cursor
// | [x, x ][1, 2, 3, 4, 5, 6, 7, 8] We need to truncate the older values and copy the newer ones
// | [old_1][ old_2 ][ <- newer -> ]
//
// # 2.2 cursor < size
//
// |            cursor
// |              v
// | [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
// | [ <- newer -> ][ <- trash  -> ]
// |              v new_cursor
// | [1, 2, 3, 4, 5, 6, 7, 8] We need to truncate the older values and copy the newer ones
// | [ <- newer -> ]
func (q *BoundedHistory[T]) SetSize(new_size int) {
	new_size = max(new_size, 1)
	if new_size != q.size {
		// resize the queue
		prev_queue := q.queue
		prev_size := q.size
		q.queue = make([]T, new_size)
		q.size = new_size
		i := q.cursor % prev_size

		// fmt.Println("Resizing history:", prev_size, "=>", new_size)
		// check if the cursor is greater than the previous size
		// if so, that means that all values within the slice have been
		// written with valid data, so we need to copy the values
		written := 0
		if q.cursor > prev_size {
			// after the cursor is the older values
			prev_older_buffer_size := prev_size - i
			new_older_buffer_size := new_size - i
			actual_size := min(new_older_buffer_size, prev_older_buffer_size)
			if actual_size > 0 {
				offset := prev_older_buffer_size - actual_size
				// fmt.Println("Offset:", offset, "actual_size:", actual_size)
				older_values := prev_queue[i+1:][offset:]

				// fmt.Println("We have already done a cycle so values after the cursor are older values")
				// fmt.Println("Cursor is at", i, "and we have", prev_size, "values, so we need to copy", actual_size, "values")
				// fmt.Println("Older values:", older_values)
				// fmt.Println("We copy them to at [0:actual_size]")
				copy(q.queue[:actual_size], older_values)
				written = actual_size - 1
				// fmt.Println("We have written", written, "older values")
			} else {
				// fmt.Println("We have no space to copy older values, skipping")
			}
		}
		// before the cursor
		new_values_size := min(new_size, i+1)
		// fmt.Println("i =", i, "new_values_size =", new_values_size)
		// fmt.Println("We have to copy", new_values_size, "values")
		newer_values_offset := i - new_values_size + 1
		newer_values := prev_queue[newer_values_offset : newer_values_offset+new_values_size]
		// fmt.Println("We copy the newer values to [written:written+i]:", newer_values)
		copy(q.queue[written:written+new_values_size], newer_values)
		written += i
		q.cursor = written
	}
}
