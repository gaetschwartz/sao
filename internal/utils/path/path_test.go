package path

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockFileSystem struct {
	FileSystem

	filesystem map[string]int
	requested  []string
}

func (f *MockFileSystem) Stat(name string) (int, error) {
	name = filepath.Clean(name)
	log.Println("Statting", name)

	f.requested = append(f.requested, name)

	if info, ok := f.filesystem[name]; ok {
		log.Println("Statting", name, "=>", info)
		return info, nil
	}
	log.Println("Statting", name, "=>", os.ErrNotExist)
	return 0, os.ErrNotExist
}

func Logger() *slog.Logger {
	opts := slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}
	handler := slog.NewTextHandler(os.Stderr, &opts)
	logger := slog.New(handler)
	return logger
}

func TestEvaluateSimple(t *testing.T) {
	evaluator := &PathPatternEvaluator{
		pattern: "{env.HOME}/devcleaner-go",
		root:    "",
		context: &PathContext{os: "darwin", arch: "amd64", environment: map[string]string{
			"HOME": "/Users/gaetan",
			"UWU":  "owo",
		}},
		l: Logger(),
		filesystem: &MockFileSystem{
			filesystem: MockFileSystemMap(
				"/Users/gaetan/devcleaner-go",
			),
		},
		lifecycle: &DefaultRuntime{},
	}
	result, err := evaluator.Evaluate()
	assert.NoError(t, err)
	assert.Equal(t, "/Users/gaetan/devcleaner-go", result)
	assert.Equal(t, []string{"/Users/gaetan", "/Users/gaetan/devcleaner-go"}, evaluator.filesystem.(*MockFileSystem).requested)
}

func TestEvaluateEither(t *testing.T) {
	evaluator := &PathPatternEvaluator{
		pattern: "[{env.HOME}/devcleaner-go-2,{env.HOME}/devcleaner-go]",
		root:    "",
		context: &PathContext{os: "darwin", arch: "amd64", environment: map[string]string{
			"HOME": "/Users/gaetan",
			"UWU":  "owo",
		}},
		l: Logger(),
		filesystem: &MockFileSystem{
			filesystem: MockFileSystemMap(
				"/Users/gaetan/devcleaner-go",
			),
		},
		lifecycle: &DefaultRuntime{},
	}
	result, err := evaluator.Evaluate()
	assert.NoError(t, err)
	assert.Equal(t, "/Users/gaetan/devcleaner-go", result)
	assert.Equal(t, []string{
		"/Users/gaetan",
		"/Users/gaetan/devcleaner-go-2",
		"/Users/gaetan",
		"/Users/gaetan/devcleaner-go",
		"/Users/gaetan/devcleaner-go", // not sure why
	},
		evaluator.filesystem.(*MockFileSystem).requested,
	)
}

func MockFileSystemMap(files ...string) map[string]int {
	m := make(map[string]int)
	for _, f := range files {
		for dir := f; dir != "/"; dir = filepath.ToSlash(filepath.Dir(dir)) {
			m[dir] = 0
		}
	}
	return m
}
