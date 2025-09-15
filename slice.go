package gx

import (
	"errors"
	"math/rand"
)

// SliceMap returns a new slice by mapping fn over in (type-changing OK).
func SliceMap[T any, R any](in []T, fn func(T) R) []R {
	out := make([]R, len(in))
	for i, v := range in {
		out[i] = fn(v)
	}
	return out
}

// SliceFilter keeps elements where fn returns true.
func SliceFilter[T any](in []T, fn func(T) bool) []T {
	out := make([]T, 0, len(in))
	for _, v := range in {
		if fn(v) {
			out = append(out, v)
		}
	}
	return out
}

// SliceReduce folds in into a single value.
func SliceReduce[T any, R any](in []T, acc R, fn func(R, T) R) R {
	cur := acc
	for _, v := range in {
		cur = fn(cur, v)
	}
	return cur
}

// SliceAny reports whether any element satisfies fn.
func SliceAny[T any](in []T, fn func(T) bool) bool {
	for _, v := range in {
		if fn(v) {
			return true
		}
	}
	return false
}

// SliceAll reports whether all elements satisfy fn.
func SliceAll[T any](in []T, fn func(T) bool) bool {
	for _, v := range in {
		if !fn(v) {
			return false
		}
	}
	return true
}

// SliceReverse returns a reversed copy.
func SliceReverse[T any](in []T) []T {
	out := make([]T, len(in))
	copy(out, in)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// SliceChunk splits into chunks of size n; returns error if n<=0.
func SliceChunk[T any](in []T, n int) ([][]T, error) {
	if n <= 0 {
		return nil, errors.New("gx.SliceChunk: n must be > 0")
	}
	var out [][]T
	for i := 0; i < len(in); i += n {
		end := i + n
		if end > len(in) {
			end = len(in)
		}
		out = append(out, in[i:end])
	}
	return out, nil
}

// SliceFlatten flattens a slice of slices.
func SliceFlatten[T any](in [][]T) []T {
	total := 0
	for _, s := range in {
		total += len(s)
	}
	out := make([]T, 0, total)
	for _, s := range in {
		out = append(out, s...)
	}
	return out
}

// SliceUniqueBy removes duplicates using a comparable key (order preserved).
func SliceUniqueBy[T any, K comparable](in []T, key func(T) K) []T {
	seen := make(map[K]struct{}, len(in))
	out := make([]T, 0, len(in))
	for _, v := range in {
		k := key(v)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, v)
	}
	return out
}

// SliceUniqueC removes duplicates for comparable T (order preserved).
func SliceUniqueC[T comparable](in []T) []T {
	seen := make(map[T]struct{}, len(in))
	out := make([]T, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// SliceUnique removes duplicates using a custom equality (order preserved, O(n^2)).
func SliceUnique[T any](in []T, equal func(a, b T) bool) []T {
	out := make([]T, 0, len(in))
	for _, v := range in {
		dup := false
		for _, u := range out {
			if equal(u, v) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, v)
		}
	}
	return out
}

// SliceGroupBy groups values by key fn.
func SliceGroupBy[T any, K comparable](in []T, keyFn func(T) K) map[K][]T {
	out := make(map[K][]T)
	for _, v := range in {
		k := keyFn(v)
		out[k] = append(out[k], v)
	}
	return out
}

// SliceIndexOf finds the first index of target using custom equality; -1 if not found.
func SliceIndexOf[T any](in []T, target T, equal func(a, b T) bool) int {
	for i, v := range in {
		if equal(v, target) {
			return i
		}
	}
	return -1
}

// SliceContains tests membership using custom equality.
func SliceContains[T any](in []T, target T, equal func(a, b T) bool) bool {
	return SliceIndexOf(in, target, equal) >= 0
}

// SliceIndexOfC finds index for comparable T; -1 if not found.
func SliceIndexOfC[T comparable](in []T, target T) int {
	for i, v := range in {
		if v == target {
			return i
		}
	}
	return -1
}

// SliceContainsC tests membership for comparable T.
func SliceContainsC[T comparable](in []T, target T) bool {
	return SliceIndexOfC(in, target) >= 0
}

// SlicePartition splits into (keep, drop) based on predicate (order preserved).
func SlicePartition[T any](in []T, fn func(T) bool) (keep []T, drop []T) {
	keep = make([]T, 0, len(in))
	drop = make([]T, 0, len(in))
	for _, v := range in {
		if fn(v) {
			keep = append(keep, v)
		} else {
			drop = append(drop, v)
		}
	}
	return
}

// SliceFind returns the first element matching fn and a boolean found flag.
func SliceFind[T any](in []T, fn func(T) bool) (T, bool) {
	for _, v := range in {
		if fn(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// SliceFindIndex returns the index of the first element matching fn; -1 if none.
func SliceFindIndex[T any](in []T, fn func(T) bool) int {
	for i, v := range in {
		if fn(v) {
			return i
		}
	}
	return -1
}

// SliceWindow creates sliding windows of size `size` with step `step`.
// Returns error if size<=0 or step<=0.
func SliceWindow[T any](in []T, size, step int) ([][]T, error) {
	if size <= 0 || step <= 0 {
		return nil, errors.New("gx.SliceWindow: size and step must be > 0")
	}
	if len(in) == 0 || len(in) < size {
		return nil, nil
	}
	out := make([][]T, 0, 1+((len(in)-size)/step))
	for i := 0; i+size <= len(in); i += step {
		out = append(out, in[i:i+size])
	}
	return out, nil
}

// SliceFindLast returns the last element matching fn.
func SliceFindLast[T any](in []T, fn func(T) bool) (T, bool) {
	for i := len(in) - 1; i >= 0; i-- {
		if fn(in[i]) {
			return in[i], true
		}
	}
	var zero T
	return zero, false
}

// SliceFindLastIndex returns the last index matching fn; -1 if none.
func SliceFindLastIndex[T any](in []T, fn func(T) bool) int {
	for i := len(in) - 1; i >= 0; i-- {
		if fn(in[i]) {
			return i
		}
	}
	return -1
}

// SliceInsert inserts v at index i (clamped). Returns new slice.
func SliceInsert[T any](in []T, i int, v T) []T {
	if i < 0 {
		i = 0
	}
	if i > len(in) {
		i = len(in)
	}
	out := make([]T, 0, len(in)+1)
	out = append(out, in[:i]...)
	out = append(out, v)
	out = append(out, in[i:]...)
	return out
}

// SliceDelete removes element at index i (if valid). Returns new slice.
func SliceDelete[T any](in []T, i int) []T {
	if i < 0 || i >= len(in) {
		return in
	}
	out := make([]T, 0, len(in)-1)
	out = append(out, in[:i]...)
	out = append(out, in[i+1:]...)
	return out
}

// SliceReplace replaces all elements matching fn with newV.
func SliceReplace[T any](in []T, fn func(T) bool, newV T) []T {
	out := make([]T, len(in))
	for i, v := range in {
		if fn(v) {
			out[i] = newV
		} else {
			out[i] = v
		}
	}
	return out
}

// SliceSplit splits into two slices at index i (clamped).
func SliceSplit[T any](in []T, i int) ([]T, []T) {
	if i < 0 {
		i = 0
	}
	if i > len(in) {
		i = len(in)
	}
	return in[:i], in[i:]
}

// SliceJoin concatenates multiple slices.
func SliceJoin[T any](slices ...[]T) []T {
	total := 0
	for _, s := range slices {
		total += len(s)
	}
	out := make([]T, 0, total)
	for _, s := range slices {
		out = append(out, s...)
	}
	return out
}

// SliceShuffle shuffles slice in place.
func SliceShuffle[T any](in []T) {
	for i := len(in) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		in[i], in[j] = in[j], in[i]
	}
}

// SliceSample picks n random elements (without replacement).
// If n > len(in), returns shuffled copy of in.
func SliceSample[T any](in []T, n int) []T {
	if n <= 0 {
		return nil
	}
	cp := make([]T, len(in))
	copy(cp, in)
	SliceShuffle(cp)
	if n > len(cp) {
		n = len(cp)
	}
	return cp[:n]
}

// SliceIntersect returns intersection of two slices (order by a).
func SliceIntersect[T comparable](a, b []T) []T {
	mb := make(map[T]struct{}, len(b))
	for _, v := range b {
		mb[v] = struct{}{}
	}
	var out []T
	for _, v := range a {
		if _, ok := mb[v]; ok {
			out = append(out, v)
		}
	}
	return SliceUniqueC(out)
}

// SliceUnion returns union of two slices (deduped).
func SliceUnion[T comparable](a, b []T) []T {
	return SliceUniqueC(append(a, b...))
}

// SliceDiff returns elements in a but not in b.
func SliceDiff[T comparable](a, b []T) []T {
	mb := make(map[T]struct{}, len(b))
	for _, v := range b {
		mb[v] = struct{}{}
	}
	var out []T
	for _, v := range a {
		if _, ok := mb[v]; !ok {
			out = append(out, v)
		}
	}
	return out
}
