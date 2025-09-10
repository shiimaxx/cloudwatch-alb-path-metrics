package main

import (
	"errors"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// filterEnv represents the environment variables available in filter expressions
type filterEnv struct {
	Method   string  `expr:"method"`   // HTTP method (GET, POST, etc.)
	Host     string  `expr:"host"`     // Host header value
	Path     string  `expr:"path"`     // Request path
	Status   int     `expr:"status"`   // HTTP status code
	Duration float64 `expr:"duration"` // Response time in seconds
}

// Contains provides string contains functionality for filter expressions
func (filterEnv) Contains(str, substr string) bool {
	return strings.Contains(str, substr)
}

// filter represents a compiled filter expression
type filter struct {
	program *vm.Program
	enabled bool
}

// newFilter creates a new filter instance with compiled expression
func newFilter(filterExpr string) (*filter, error) {
	// Empty filter means no filtering
	if filterExpr == "" {
		return &filter{enabled: false}, nil
	}

	// Create a sample environment for compilation
	env := filterEnv{}

	// Compile the expression with the environment context
	program, err := expr.Compile(filterExpr, expr.Env(env), expr.AsBool())
	if err != nil {
		return nil, errors.New("failed to compile filter expression: " + err.Error())
	}

	return &filter{
		program: program,
		enabled: true,
	}, nil
}

// evaluate evaluates the compiled filter against an ALB log entry
// Returns true if the entry passes the filter, false otherwise
func (f *filter) evaluate(entry albLogEntry) (bool, error) {
	// If filter is disabled, allow all entries
	if !f.enabled {
		return true, nil
	}

	// Create the environment for the expression evaluation
	env := filterEnv{
		Method:   entry.method,
		Host:     entry.host,
		Path:     entry.path,
		Status:   entry.status,
		Duration: entry.duration,
	}

	// Execute the compiled expression
	result, err := vm.Run(f.program, env)
	if err != nil {
		return false, errors.New("failed to evaluate filter expression: " + err.Error())
	}

	return result.(bool), nil
}
