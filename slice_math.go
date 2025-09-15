package gx

import (
	"cmp"
	"errors"
	"math"
	"slices"
	"sort"

	"golang.org/x/exp/constraints"
)

// ------------------- Sum / Avg -------------------

// SliceSum adds up numeric slice elements.
// Works for any integer or float.
func SliceSum[T constraints.Integer | constraints.Float](in []T) T {
	var sum T
	for _, v := range in {
		sum += v
	}
	return sum
}

// SliceAvg computes average value. Returns error if slice is empty.
func SliceAvg[T constraints.Integer | constraints.Float](in []T) (float64, error) {
	if len(in) == 0 {
		return 0, errors.New("gx.SliceAvg: empty slice")
	}
	var sum T
	for _, v := range in {
		sum += v
	}
	return float64(sum) / float64(len(in)), nil
}

// ------------------- Min / Max -------------------

// SliceMin returns the smallest element. Errors if empty.
func SliceMin[T cmp.Ordered](in []T) (T, error) {
	if len(in) == 0 {
		var zero T
		return zero, errors.New("gx.SliceMin: empty slice")
	}
	min := in[0]
	for _, v := range in[1:] {
		if v < min {
			min = v
		}
	}
	return min, nil
}

// SliceMax returns the largest element. Errors if empty.
func SliceMax[T cmp.Ordered](in []T) (T, error) {
	if len(in) == 0 {
		var zero T
		return zero, errors.New("gx.SliceMax: empty slice")
	}
	max := in[0]
	for _, v := range in[1:] {
		if v > max {
			max = v
		}
	}
	return max, nil
}

// SliceMinMax returns both min and max. Errors if empty.
func SliceMinMax[T cmp.Ordered](in []T) (T, T, error) {
	if len(in) == 0 {
		var zero T
		return zero, zero, errors.New("gx.SliceMinMax: empty slice")
	}
	min, max := in[0], in[0]
	for _, v := range in[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max, nil
}

// ------------------------ Median ------------------------

// SliceMedian returns the median as float64.
// For even-length slices it averages the two middle values.
// Returns error on empty input.
func SliceMedian[T constraints.Integer | constraints.Float](in []T) (float64, error) {
	n := len(in)
	if n == 0 {
		return 0, errors.New("gx.SliceMedian: empty slice")
	}
	cp := make([]T, n)
	copy(cp, in)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	mid := n / 2
	if n%2 == 1 {
		return float64(cp[mid]), nil
	}
	// even: average the two middle values
	return (float64(cp[mid-1]) + float64(cp[mid])) / 2.0, nil
}

// ------------------------ Percentile ------------------------

// SlicePercentile computes the p-th percentile (0..100) using
// linear interpolation between closest ranks (a.k.a. type-7 in many libs).
// Returns error on empty input or out-of-range p.
func SlicePercentile[T constraints.Integer | constraints.Float](in []T, p float64) (float64, error) {
	n := len(in)
	if n == 0 {
		return 0, errors.New("gx.SlicePercentile: empty slice")
	}
	if p < 0 || p > 100 || math.IsNaN(p) {
		return 0, errors.New("gx.SlicePercentile: p must be within [0,100]")
	}
	if n == 1 {
		return float64(in[0]), nil
	}
	cp := make([]T, n)
	copy(cp, in)
	slices.Sort(cp)

	// Linear interpolation on index i = p/100 * (n-1)
	pos := (p / 100.0) * float64(n-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return float64(cp[lo]), nil
	}
	frac := pos - float64(lo)
	return (1-frac)*float64(cp[lo]) + frac*float64(cp[hi]), nil
}

// ------------------------ Variance / StdDev ------------------------

// SliceVariance computes variance using Welford's online algorithm.
// If sample==true, divides by (n-1); else divides by n (population).
// Returns error on empty input or sample with n<2.
func SliceVariance[T constraints.Integer | constraints.Float](in []T, sample bool) (float64, error) {
	n := len(in)
	if n == 0 {
		return 0, errors.New("gx.SliceVariance: empty slice")
	}
	if sample && n < 2 {
		return 0, errors.New("gx.SliceVariance: need at least 2 values for sample variance")
	}
	var mean float64
	var m2 float64 // sum of squared deviations
	var k float64
	for _, v := range in {
		k++
		x := float64(v)
		delta := x - mean
		mean += delta / k
		m2 += delta * (x - mean)
	}
	if sample {
		return m2 / float64(n-1), nil
	}
	return m2 / float64(n), nil
}

// SliceStdDev returns sqrt of SliceVariance (sample or population).
func SliceStdDev[T constraints.Integer | constraints.Float](in []T, sample bool) (float64, error) {
	variance, err := SliceVariance(in, sample)
	if err != nil {
		return 0, err
	}
	return math.Sqrt(variance), nil
}
