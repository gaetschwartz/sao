package path

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// A pattern is a path-like string that can contains different patterns:
//   - variables
//   - placeholders
//   - either
//
// # Variables
//
// Variables are identifiers that can be used to access environment variables or app-specific variables
// Variables are surrounded by curly braces and take the form of
// `{<variable>}`
//
// Examples:
//   - {os}
//   - {arch}
//   - {app_path}
//   - {env.HOME}, {env.PATH}, etc.
//
// # Placeholders
//
// Placeholders take the shape of
// `{<cond1>:<value1>,<cond2>:<value2>,<default?>}`
// A condition is an expression that evaluates to a boolean i.e
//
// > cond1 := expr1 op expr2
//
// where op is one of `==`, `!=`, `<`, `>`, `<=`, `>=`
// where expr1 and expr2 are either:
//   - a string
//   - a number
//   - a variable
//
// If no default is specified, and none of the conditions are met, the pattern is empty
// In instead it is desired to throw an error, the default can be specified as follows:
// {<cond1>:<value1>,<cond2>:<value2>,!}
// If a path contains one of these characters: '{', '}', they have to be escaped with a backslash
// Paths inside a placeholder are evaluated as a path pattern and should be surrounded by curly braces
// if it contains any characters susceptible to being interpreted as a placeholder.
//
// Examples
//
//   - {os==darwin:/usr/local/bin/brew,linux:/home/linuxbrew/.linuxbrew/bin/brew}
//   - {env.HOME}/.cargo/bin
//   - {app_path}/node_modules/.bin
//
// # Either
//
// Either is a pattern that can be used to specify one or more alternatives to be tried if the first path does not exist
// Either is surrounded by square brackets and takes the form of
// `[<path1>,<path2>,...]`
//
// Examples:
//   - [{env.HOME}/.cargo/bin, {env.CARGO_HOME}/bin]
type PathPattern string

type PathContext struct {
	environment map[string]string
	appPath     string
	os          string
	arch        string
}

func (p PathPattern) Eval(context *PathContext) (string, error) {
	handler := slog.NewTextHandler(os.Stderr, nil)
	logger := slog.New(handler)
	eval := &PathPatternEvaluator{
		pattern:    string(p),
		root:       "",
		context:    context,
		l:          logger,
		filesystem: &RealFileSystem{},
		lifecycle:  &DefaultRuntime{},
	}

	return eval.Evaluate()
}

func NewPathContext() *PathContext {
	environ_array := os.Environ()
	environ := make(map[string]string)
	for _, env := range environ_array {
		parts := strings.SplitN(env, "=", 2)
		environ[parts[0]] = parts[1]
	}
	return &PathContext{
		environment: environ,
		appPath:     "",
		os:          runtime.GOOS,
		arch:        runtime.GOARCH,
	}
}

func (c *PathContext) GetEnv(name string) string {
	return c.environment[name]
}

type FileSystem interface {
	Stat(string) (int, error)
}

type RealFileSystem struct {
	FileSystem
}

func (f *RealFileSystem) Stat(name string) (int, error) {
	if val, err := os.Stat(name); err == nil {
		return int(val.Size()), nil
	} else {
		return 0, err
	}
}

type PathPatternEvaluator struct {
	pattern    string
	root       string
	context    *PathContext
	l          *slog.Logger
	filesystem FileSystem
	lifecycle  Runtime
}

func NewPathPatternEvaluator(pattern string) *PathPatternEvaluator {
	opts := slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}
	handler := slog.NewTextHandler(os.Stderr, &opts)
	logger := slog.New(handler)
	return &PathPatternEvaluator{
		pattern:    pattern,
		root:       "",
		context:    NewPathContext(),
		l:          logger,
		filesystem: &RealFileSystem{},
		lifecycle:  &DefaultRuntime{},
	}
}

func (p *PathPatternEvaluator) Evaluate() (string, error) {
	p.l.Debug("Evaluating pattern", "pattern", p.pattern)
	result, err := p.evaluateInternal(p.pattern)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Clean(result)), nil
}

func (p *PathPatternEvaluator) evaluateInternal(pattern string) (string, error) {
	var result strings.Builder

	for len(pattern) > 0 {
		switch pattern[0] {
		case '{':
			p.l.Debug("Evaluating variable", "pattern", pattern)
			closingBrace := findClosingBrace(pattern)
			if closingBrace == -1 {
				return "", errors.New("unmatched opening brace")
			}
			subPattern := pattern[1:closingBrace]
			evaluated, err := p.evaluateVariable(subPattern)
			if err != nil {
				return "", fmt.Errorf("error evaluating variable %s (%w)", subPattern, err)
			}
			evaluated.WriteToBuilder(&result)
			pattern = pattern[closingBrace+1:]

			p.l.Debug("Evaluated variable", "pattern", pattern, "evaluated", evaluated)
		case '[':
			p.l.Debug("Evaluating either", "pattern", pattern)
			closingBracket := findClosingBracket(pattern)
			if closingBracket == -1 {
				return "", errors.New("unmatched opening bracket")
			}
			subPattern := pattern[1:closingBracket]
			evaluated, err := p.evaluateEither(subPattern)
			if err != nil {
				return "", err
			}
			p.l.Debug("Evaluated either", "pattern", pattern, "evaluated", evaluated)

			evaluated.WriteToBuilder(&result)
			pattern = pattern[closingBracket+1:]
		case '\\':
			p.l.Debug("Evaluating escape sequence", "pattern", pattern)
			if len(pattern) > 1 {
				result.WriteByte(pattern[1])
				pattern = pattern[2:]
			} else {
				return "", errors.New("invalid escape sequence")
			}
		default:
			result.WriteByte(pattern[0])
			pattern = pattern[1:]
		}
	}

	out := result.String()
	// check if the result is a valid path
	if _, err := p.filesystem.Stat(out); err != nil {
		return "", fmt.Errorf("invalid path %s (%w)", out, err)
	}
	return out, nil
}

func (p *PathPatternEvaluator) evaluateVariable(variable string) (ExistingPath, error) {
	// Implement variable evaluation logic here
	// This should handle environment variables, app-specific variables, and placeholders
	var result string
	if strings.HasPrefix(variable, "env.") {
		envVar := variable[4:]
		value := p.context.GetEnv(envVar)
		if value == "" {
			return "", fmt.Errorf("environment variable %s not found", envVar)
		}
		result = value
	} else {
		switch variable {
		case "os":
			result = p.context.os
		case "arch":
			result = p.context.arch
		case "app_path":
			if p.context.appPath == "" {
				return "", errors.New("app_path not set")
			} else {
				result = p.context.appPath
			}
		default:
			return "", fmt.Errorf("unknown variable %s", variable)
		}
	}

	return p.Exists(result)
}

type ExistingPath string

func (p *PathPatternEvaluator) Exists(subpath string) (ExistingPath, error) {
	if !p.lifecycle.AllowedToRun() {
		fmt.Println("PathPatternEvaluator.Exists(): lifecycle manager killed us :(")
		return "", errors.New("lifecycle manager killed us :(")
	}
	fullPath := filepath.Join(p.root, subpath)
	if _, err := p.filesystem.Stat(fullPath); err != nil {
		return "", fmt.Errorf("path %s does not exist (%w)", fullPath, err)
	}
	return ExistingPath(fullPath), nil
}

func (p *ExistingPath) WriteToBuilder(builder *strings.Builder) {
	builder.WriteString(string(*p))
}

type EvalEitherFunc func(string) (ExistingPath, error)

func (p *PathPatternEvaluator) evaluateEither(either string) (ExistingPath, error) {
	options, err := SplitCommaSeparatedString(either)
	if err != nil {
		return "", err
	}
	p.l.Debug("Split either", "either", either, "options", options)
	Task := func(_ int, opt string) (ExistingPath, error) {
		subEvaluator := &PathPatternEvaluator{
			pattern:    opt,
			root:       p.root,
			context:    p.context,
			l:          p.l,
			filesystem: p.filesystem,
			lifecycle:  p.lifecycle,
		}
		p.l.Debug("Evaluating either option", "option", opt)
		if evaluated, err := subEvaluator.Evaluate(); err == nil {
			p.l.Debug("Found valid path in either", "path", evaluated)
			return ExistingPath(evaluated), nil
		} else {
			p.l.Debug("Skipping invalid path in either", "option", opt, "path", evaluated)
			return "", err
		}
	}
	res := p.lifecycle.RunAll(Task, options)
	// p.l.Debug("Found results", "results", res)
	return FirstResult(res)
}

func findClosingBrace(s string) int {
	depth := 0
	for i, ch := range s {
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findClosingBracket(s string) int {
	depth := 0
	for i, ch := range s {
		if ch == '[' {
			depth++
		} else if ch == ']' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func SplitCommaSeparatedString(s string) ([]string, error) {
	// We need to take in account that fact that commas could be escaed or nested in another pattern
	// For example:
	//   [/path1,{env.HOME}/[path2,path3],/path4]
	// should be evaluated as:
	//   /path1
	//   {env.HOME}/path2
	//   {env.HOME}/path3
	//   /path4
	var result []string
	i := 0
	chunk_start := 0
	for ; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case ',':
			result = append(result, s[chunk_start:i])
			i++
			chunk_start = i
		case '[':
			closingBracket := findClosingBracket(s[i:])
			if closingBracket == -1 {
				return nil, errors.New("unmatched opening bracket")
			}
			i += closingBracket + 1
		case '{':
			closingBrace := findClosingBrace(s[i:])
			if closingBrace == -1 {
				return nil, errors.New("unmatched opening brace")
			}
			i += closingBrace + 1
		}
	}
	rest := s[chunk_start:]
	if len(rest) > 0 {
		result = append(result, rest)
	}
	return result, nil
}
