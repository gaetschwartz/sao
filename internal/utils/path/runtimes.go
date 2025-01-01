package path

import (
	"fmt"
	"sync"
)

type Runtime interface {
	LifecycleManager[string, ExistingPath]
}

type DefaultRuntime struct {
	SynchronousLifecycleManager[string, ExistingPath]
}

type GoRoutinesRuntime struct {
	GoRoutinesLifecycleManager[string, ExistingPath]
	quit chan int
}

type LifecycleManager[T, R any] interface {
	AllowedToRun() bool
	RunAll(func(int, T) (R, error), []T) []TaskResult[R]
}

type SynchronousLifecycleManager[T, R any] struct {
	LifecycleManager[T, R]
}

func (m *SynchronousLifecycleManager[T, R]) AllowedToRun() bool {
	return true
}

type TaskResult[T any] struct {
	Result T
	Err    error
}

func FirstResult[T any](t []TaskResult[T]) (T, error) {
	for _, r := range t {
		if r.Err == nil {
			return r.Result, nil
		}
	}

	last := t[len(t)-1]
	return last.Result, last.Err
}

func WithoutError[T any](t []TaskResult[T]) ([]T, error) {
	results := make([]T, len(t))
	for i, r := range t {
		if r.Err == nil {
			results[i] = r.Result
		} else {
			return nil, r.Err
		}
	}
	return results, nil
}

func (m *SynchronousLifecycleManager[T, R]) RunAll(f func(int, T) (R, error), elems []T) []TaskResult[R] {
	// fmt.Println("Running synchronously")
	results := make([]TaskResult[R], len(elems))
	for i, elem := range elems {
		if result, err := f(i, elem); err == nil {
			results[i] = TaskResult[R]{Result: result}
		} else {
			results[i] = TaskResult[R]{Err: err}
		}
	}
	return results
}

type GoRoutinesLifecycleManager[T, R any] struct {
	quit chan int
}

func (m *GoRoutinesLifecycleManager[T, R]) AllowedToRun() bool {
	fmt.Println("GoRoutinesLifecycleManager.AllowedToRun()")
	select {
	case c := <-m.quit:
		fmt.Println("GoRoutinesLifecycleManager.AllowedToRun(): quit", c)
		return false
	default:
		return true
	}
}

func (m *GoRoutinesLifecycleManager[T, R]) RunAll(f func(int, T) (R, error), elems []T) []TaskResult[R] {
	// fmt.Println("Running in goroutines")
	ch := make(chan R, len(elems))
	defer close(ch)
	err_ch := make(chan error, len(elems))
	defer close(err_ch)
	var wg sync.WaitGroup
	wg.Add(len(elems))
	for i, elem := range elems {
		go func(i int, elem T) {
			defer wg.Done()
			if result, err := f(i, elem); err == nil {
				ch <- result
				err_ch <- nil
			} else {
				ch <- result
				err_ch <- err
			}
		}(i, elem)
	}
	wg.Wait()

	results := make([]TaskResult[R], len(elems))
	for i := range len(elems) {
		if res, err := <-ch, <-err_ch; err == nil {
			results[i] = TaskResult[R]{Result: res}
		} else {
			results[i] = TaskResult[R]{Err: err}
		}
	}
	return results
}
