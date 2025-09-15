package gx

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// PerfNow returns current high-resolution timestamp.
func PerfNow() time.Time { return time.Now() }

// PerfSince returns duration since t, as time.Duration.
func PerfSince(t time.Time) time.Duration { return time.Since(t) }

// PerfTrack prints how long a function took. Use with defer:
//
//	defer gx.PerfTrack("myFunc", gx.PerfNow())
func PerfTrack(name string, start time.Time) {
	d := time.Since(start)
	fmt.Fprintf(os.Stderr, "[PERF] %s took %s\n", name, d)
}

// PerfMeasure runs fn n times, returns average duration and total duration.
func PerfMeasure(n int, fn func()) (avg, total time.Duration) {
	if n <= 0 {
		return 0, 0
	}
	start := time.Now()
	for range n {
		fn()
	}
	total = time.Since(start)
	avg = total / time.Duration(n)
	return
}

// PerfMemStats returns current allocated bytes and total system bytes.
func PerfMemStats() (alloc, sys uint64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc, m.Sys
}

// PerfGCForces triggers GC and returns duration it took.
func PerfGCForces() time.Duration {
	start := time.Now()
	runtime.GC()
	return time.Since(start)
}

// PerfThroughput runs fn n times and returns ops/sec.
func PerfThroughput(n int, fn func()) float64 {
	if n <= 0 {
		return 0
	}
	start := time.Now()
	for range n {
		fn()
	}
	elapsed := time.Since(start).Seconds()
	return float64(n) / elapsed
}

// PerfCase describes a function to benchmark.
type PerfCase struct {
	Name string
	N    int    // number of iterations
	Fn   func() // function to run
}

// PerfBenchResult holds results for a benchmark case.
type PerfBenchResult struct {
	Name   string
	N      int
	Total  time.Duration
	Avg    time.Duration
	OpsSec float64
}

// PerfBenchTable runs multiple cases and prints a table.
// Returns results for programmatic use.
func PerfBenchTable(cases []PerfCase) []PerfBenchResult {
	results := make([]PerfBenchResult, 0, len(cases))
	for _, c := range cases {
		if c.N <= 0 {
			c.N = 1
		}
		start := time.Now()
		for i := 0; i < c.N; i++ {
			c.Fn()
		}
		total := time.Since(start)
		avg := total / time.Duration(c.N)
		ops := float64(c.N) / total.Seconds()
		results = append(results, PerfBenchResult{
			Name:   c.Name,
			N:      c.N,
			Total:  total,
			Avg:    avg,
			OpsSec: ops,
		})
	}

	// pretty print
	fmt.Fprintf(os.Stderr, "\n%-15s %-10s %-12s %-12s %-12s\n",
		"Name", "N", "Total", "Avg", "Ops/sec")
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("-", 65))
	for _, r := range results {
		fmt.Fprintf(os.Stderr, "%-15s %-10d %-12s %-12s %-12.0f\n",
			r.Name, r.N, r.Total, r.Avg, r.OpsSec)
	}
	fmt.Fprintln(os.Stderr)
	return results
}
