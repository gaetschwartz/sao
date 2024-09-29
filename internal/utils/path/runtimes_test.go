package path

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func F(i int, elem string) (string, error) {
	return (fmt.Sprintf("%02d-%s", i, elem)), nil
}

func FDelayed(i int, elem string) (string, error) {
	time.Sleep(time.Millisecond * 10 * time.Duration(i))
	return (fmt.Sprintf("%02d-%s", i, elem)), nil
}

// GetArray returns a slice of strings from 'a' to 'z'
func GetArray() []string {
	arr := make([]string, 26)
	for i := 0; i < len(arr); i++ {
		arr[i] = string(rune('a' + i))
	}
	return arr
}

// GetExpectedResults returns the GetArray() elements prefixed with their index
func GetExpectedResults() []string {
	arr := make([]string, 26)
	for i, elem := range GetArray() {
		arr[i] = fmt.Sprintf("%02d-%s", i, elem)
	}
	return arr
}

func TestSyncRunAll(t *testing.T) {
	m := &SynchronousLifecycleManager[string, string]{}
	results, err := m.RunAll(F, GetArray())
	assert.NoError(t, err)
	assert.Equal(t, GetExpectedResults(), results)
}

func TestGoRoutinesRunAll(t *testing.T) {
	m := &GoRoutinesLifecycleManager[string, string]{quit: make(chan int)}
	results, err := m.RunAll(F, GetArray())
	slices.Sort(results)
	assert.NoError(t, err)
	assert.Equal(t, GetExpectedResults(), results)
}

func TestGoRoutinesIsFaster(t *testing.T) {
	m := &GoRoutinesLifecycleManager[string, string]{quit: make(chan int)}
	start := time.Now()
	results, err := m.RunAll(FDelayed, GetArray())
	took1 := time.Since(start)
	assert.NoError(t, err)
	assert.Equal(t, GetExpectedResults(), results)
	fmt.Println("GoRoutines took", took1)

	m2 := &SynchronousLifecycleManager[string, string]{}
	start = time.Now()
	results, err = m2.RunAll(FDelayed, GetArray())
	took2 := time.Since(start)
	assert.NoError(t, err)
	assert.Equal(t, GetExpectedResults(), results)
	fmt.Println("Synchronous took", took2)

	faster := "GoRoutines"
	by := took2 / took1
	if took2 < took1 {
		faster = "Synchronous"
		by = took1 / took2
	}
	fmt.Println(faster, "is", int(by), "times faster")
}
