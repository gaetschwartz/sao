package history

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoundedHistoryAddingAndRestoring(t *testing.T) {
	h := NewBoundedHistory[int](3)
	h.Add(1)                    //  v cursor
	assert.Equal(t, 1, h.Get()) // [1, x, x]
	h.Add(2)                    //     v cursor
	assert.Equal(t, 2, h.Get()) // [1, 2, x]
	h.Add(3)                    //        v cursor
	assert.Equal(t, 3, h.Get()) // [1, 2, 3]
	h.Add(4)                    //  v cursor
	assert.Equal(t, 4, h.Get()) // [4, 2, 3]
	h.Add(5)                    //     v cursor
	assert.Equal(t, 5, h.Get()) // [4, 5, 3]
	el1, can1 := h.Restore()    //  v cursor
	assert.Equal(t, 4, el1)     // [4, 5, 3]
	assert.True(t, can1)        //
	el2, can2 := h.Restore()    //  			 v cursor
	assert.Equal(t, 3, el2)     // [4, 5, 3]
	assert.True(t, can2)        //
	el3, can3 := h.Restore()    //        v cursor
	assert.Equal(t, 3, el3)     // [4, 5, 3]
	assert.False(t, can3)       //        v cursor
}

func TestBoundedHistoryAddAll(t *testing.T) {
	h := NewBoundedHistory[int](3)
	h.AddAll([]int{1, 2, 3, 4, 5})
	assert.Equal(t, []int{4, 5, 3}, h.queue)
	assert.Equal(t, 4, h.cursor)
	assert.Equal(t, 5, h.written)
	assert.Equal(t, 5, h.Get())
}

func TestBoundedHistoryResizingBigger(t *testing.T) {
	h := NewBoundedHistory[int](5)
	h.AddAll([]int{1, 2, 3, 4, 5, 6, 7, 8})                 //        v cursor
	assert.Equal(t, []int{6, 7, 8, 4, 5}, h.queue)          // [6, 7, 8, 4, 5]
	h.SetSize(8)                                            //              v cursor
	assert.Equal(t, []int{4, 5, 6, 7, 8, 0, 0, 0}, h.queue) // [4, 5, 6, 7, 8, x, x, x]
}

func TestBoundedHistoryResizingSmaller(t *testing.T) {
	h := NewBoundedHistory[int](5)
	h.AddAll([]int{1, 2, 3, 4, 5, 6, 7, 8, 9})     //           v cursor
	assert.Equal(t, []int{6, 7, 8, 9, 5}, h.queue) // [6, 7, 8, 9, 5]
	h.SetSize(3)                                   //        v cursor
	assert.Equal(t, []int{7, 8, 9}, h.queue)       // [7, 8, 9]
}

func TestBoundedHistoryResizingSmaller2(t *testing.T) {
	h := NewBoundedHistory[int](5)
	h.AddAll([]int{1, 2, 3, 4, 5, 6, 7, 8, 9})     //           v cursor
	assert.Equal(t, []int{6, 7, 8, 9, 5}, h.queue) // [6, 7, 8, 9, 5]
	h.SetSize(2)                                   //        v cursor
	assert.Equal(t, []int{8, 9}, h.queue)          // [8, 9]
}
