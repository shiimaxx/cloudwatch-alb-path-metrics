package main

import (
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

var filterProgram *vm.Program

func filter(method, host, path string) bool {
	if filterProgram == nil {
		return true
	}

	env := map[string]any{
		"method": method,
		"host":   host,
		"path":   path,
	}

	result, err := expr.Run(filterProgram, env)
	if err != nil {
		fmt.Printf("failed to evaluate filter expression: %v\n", err)
		return false
	}

	return result.(bool)
}
