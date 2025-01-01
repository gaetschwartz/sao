package io

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// Recursively calculates the disk usage of a directory
// This uses goroutines to parallelize the calculation
func DiskUsage(path string) (int64, error) {
	// fmt.Println("DiskUsage", path)
	var size atomic.Int64
	var wg sync.WaitGroup
	errors := make(chan error)
	defer close(errors)

	wg.Add(1)
	go diskUsage(path, &size, errors, &wg)
	wg.Wait()
	// drain all the values from the channel
	for len(errors) > 0 {
		if err := <-errors; err != nil {
			fmt.Printf("DiskUsage(%s): error: %s\n", path, err)
			return 0, err
		}
	}

	total := size.Load()
	// fmt.Printf("DiskUsage(%s): total: %d\n", path, total)

	return total, nil
}

func diskUsage(path string, size *atomic.Int64, errors chan error, wg *sync.WaitGroup) {
	// fmt.Printf("diskUsage(%s): Starting\n", path)
	defer func() {
		if r := recover(); r != nil {
			errors <- fmt.Errorf("panic in diskUsage: %s", r)
		}
		wg.Done()
	}()

	dir, err := os.ReadDir(path)
	if err != nil {
		errors <- fmt.Errorf("error reading directory %s: %w", path, err)
		return
	}
	for _, entry := range dir {
		if entry.IsDir() {
			wg.Add(1)
			go diskUsage(filepath.Join(path, entry.Name()), size, errors, wg)
			continue
		}
		info, err := entry.Info()
		if err != nil {
			errors <- fmt.Errorf("error getting info for %s: %w", entry.Name(), err)
			continue
		}
		size.Add(info.Size())
	}
}

func HumanizeBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(size)/float64(div), "kMGTPE"[exp])
}
