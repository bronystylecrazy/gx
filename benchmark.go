package gx

import (
	"fmt"
	"log"
	"time"
)

// BenchmarkScope ensures execution time is measured for the given scope.
// If an empty string is provided, it automatically assigns an "Anonymous" label.
func BenchmarkScope(scope string) func() {
	if scope == "" {
		scope = "Anonymous"
	}
	start := time.Now() // Capture the start time when the function is called
	return func() {
		elapsed := time.Since(start) // Calculate elapsed time
		fmt.Printf("[%s] Execution Time: %s\n", scope, elapsed)
	}
}

// Benchmark measures the execution time of a function.
func Benchmark(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// BenchmarkLog measures the execution time of a function and logs the result.
func BenchmarkLog(label string, fn func()) {
	start := time.Now()
	fn()
	elapsed := time.Since(start)
	log.Printf("[%s] Execution Time: %s\n", label, elapsed)
}
