package gx

import (
	"cmp"
	"errors"
	"sort"
)

// Pair is a simple 2-tuple used by SliceZip / SliceUnzip.
type Pair[A any, B any] struct {
	A A
	B B
}

// ------------------------- Zip / Unzip -------------------------

// SliceZip pairs elements from a and b, truncating to the shorter length.
func SliceZip[A any, B any](a []A, b []B) []Pair[A, B] {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	out := make([]Pair[A, B], n)
	for i := 0; i < n; i++ {
		out[i] = Pair[A, B]{A: a[i], B: b[i]}
	}
	return out
}

// SliceZipStrict pairs elements and errors if lengths differ.
func SliceZipStrict[A any, B any](a []A, b []B) ([]Pair[A, B], error) {
	if len(a) != len(b) {
		return nil, errors.New("gx.SliceZipStrict: length mismatch")
	}
	out := make([]Pair[A, B], len(a))
	for i := range a {
		out[i] = Pair[A, B]{A: a[i], B: b[i]}
	}
	return out, nil
}

// SliceUnzip splits a slice of pairs back into two slices.
func SliceUnzip[A any, B any](in []Pair[A, B]) ([]A, []B) {
	aa := make([]A, len(in))
	bb := make([]B, len(in))
	for i, p := range in {
		aa[i], bb[i] = p.A, p.B
	}
	return aa, bb
}

// ------------------------- Sorters -------------------------

// NOTE: *By* variants return a NEW slice (non-destructive).
//       *InPlace* variants sort the given slice directly.
//       Stable versions preserve relative order of equal elements.

// SliceSortBy returns a new slice sorted by a less(a,b) comparator.
func SliceSortBy[T any](in []T, less func(a, b T) bool) []T {
	out := make([]T, len(in))
	copy(out, in)
	sort.Slice(out, func(i, j int) bool { return less(out[i], out[j]) })
	return out
}

// SliceSortByInPlace sorts the slice in place using less(a,b).
func SliceSortByInPlace[T any](in []T, less func(a, b T) bool) {
	sort.Slice(in, func(i, j int) bool { return less(in[i], in[j]) })
}

// SliceSortStableBy returns a new slice with a stable sort by less(a,b).
func SliceSortStableBy[T any](in []T, less func(a, b T) bool) []T {
	out := make([]T, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool { return less(out[i], out[j]) })
	return out
}

// SliceSortStableByInPlace stably sorts the slice in place using less(a,b).
func SliceSortStableByInPlace[T any](in []T, less func(a, b T) bool) {
	sort.SliceStable(in, func(i, j int) bool { return less(in[i], in[j]) })
}

// ------------------------- Key-based (cmp.Ordered) -------------------------

// SliceSortByKey copies and sorts by key(x) (fast path for ordered keys).
func SliceSortByKey[T any, K cmp.Ordered](in []T, key func(T) K) []T {
	out := make([]T, len(in))
	copy(out, in)
	sort.Slice(out, func(i, j int) bool { return key(out[i]) < key(out[j]) })
	return out
}

// SliceSortByKeyInPlace sorts in place by key(x).
func SliceSortByKeyInPlace[T any, K cmp.Ordered](in []T, key func(T) K) {
	sort.Slice(in, func(i, j int) bool { return key(in[i]) < key(in[j]) })
}

// SliceSortStableByKey copies and performs a stable sort by key(x).
func SliceSortStableByKey[T any, K cmp.Ordered](in []T, key func(T) K) []T {
	out := make([]T, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool { return key(out[i]) < key(out[j]) })
	return out
}

// SliceSortStableByKeyInPlace stably sorts in place by key(x).
func SliceSortStableByKeyInPlace[T any, K cmp.Ordered](in []T, key func(T) K) {
	sort.SliceStable(in, func(i, j int) bool { return key(in[i]) < key(in[j]) })
}

// ------------------------- Zip3 / Zip4 -------------------------

type Triple[A any, B any, C any] struct {
	A A
	B B
	C C
}
type Quad[A any, B any, C any, D any] struct {
	A A
	B B
	C C
	D D
}

// Truncates to shortest length
func SliceZip3[A, B, C any](a []A, b []B, c []C) []Triple[A, B, C] {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if len(c) < n {
		n = len(c)
	}
	out := make([]Triple[A, B, C], n)
	for i := 0; i < n; i++ {
		out[i] = Triple[A, B, C]{a[i], b[i], c[i]}
	}
	return out
}

// Errors if lengths differ
func SliceZip3Strict[A, B, C any](a []A, b []B, c []C) ([]Triple[A, B, C], error) {
	if len(a) != len(b) || len(a) != len(c) {
		return nil, errors.New("gx.SliceZip3Strict: length mismatch")
	}
	out := make([]Triple[A, B, C], len(a))
	for i := range a {
		out[i] = Triple[A, B, C]{a[i], b[i], c[i]}
	}
	return out, nil
}

func SliceUnzip3[A, B, C any](in []Triple[A, B, C]) ([]A, []B, []C) {
	aa := make([]A, len(in))
	bb := make([]B, len(in))
	cc := make([]C, len(in))
	for i, t := range in {
		aa[i], bb[i], cc[i] = t.A, t.B, t.C
	}
	return aa, bb, cc
}

// -------- Zip4 --------

func SliceZip4[A, B, C, D any](a []A, b []B, c []C, d []D) []Quad[A, B, C, D] {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if len(c) < n {
		n = len(c)
	}
	if len(d) < n {
		n = len(d)
	}
	out := make([]Quad[A, B, C, D], n)
	for i := 0; i < n; i++ {
		out[i] = Quad[A, B, C, D]{a[i], b[i], c[i], d[i]}
	}
	return out
}

func SliceZip4Strict[A, B, C, D any](a []A, b []B, c []C, d []D) ([]Quad[A, B, C, D], error) {
	if len(a) != len(b) || len(a) != len(c) || len(a) != len(d) {
		return nil, errors.New("gx.SliceZip4Strict: length mismatch")
	}
	out := make([]Quad[A, B, C, D], len(a))
	for i := range a {
		out[i] = Quad[A, B, C, D]{a[i], b[i], c[i], d[i]}
	}
	return out, nil
}

func SliceUnzip4[A, B, C, D any](in []Quad[A, B, C, D]) ([]A, []B, []C, []D) {
	aa := make([]A, len(in))
	bb := make([]B, len(in))
	cc := make([]C, len(in))
	dd := make([]D, len(in))
	for i, q := range in {
		aa[i], bb[i], cc[i], dd[i] = q.A, q.B, q.C, q.D
	}
	return aa, bb, cc, dd
}

// ------------------------- SortBy* DESC sugar -------------------------

// Comparator-based (copy)
func SliceSortByDesc[T any](in []T, less func(a, b T) bool) []T {
	out := make([]T, len(in))
	copy(out, in)
	sort.Slice(out, func(i, j int) bool { return less(out[j], out[i]) })
	return out
}
func SliceSortByDescInPlace[T any](in []T, less func(a, b T) bool) {
	sort.Slice(in, func(i, j int) bool { return less(in[j], in[i]) })
}
func SliceSortStableByDesc[T any](in []T, less func(a, b T) bool) []T {
	out := make([]T, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool { return less(out[j], out[i]) })
	return out
}
func SliceSortStableByDescInPlace[T any](in []T, less func(a, b T) bool) {
	sort.SliceStable(in, func(i, j int) bool { return less(in[j], in[i]) })
}

// Key-based (cmp.Ordered) (copy)
func SliceSortByKeyDesc[T any, K cmp.Ordered](in []T, key func(T) K) []T {
	out := make([]T, len(in))
	copy(out, in)
	sort.Slice(out, func(i, j int) bool { return key(out[j]) < key(out[i]) })
	return out
}
func SliceSortByKeyDescInPlace[T any, K cmp.Ordered](in []T, key func(T) K) {
	sort.Slice(in, func(i, j int) bool { return key(in[j]) < key(in[i]) })
}
func SliceSortStableByKeyDesc[T any, K cmp.Ordered](in []T, key func(T) K) []T {
	out := make([]T, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool { return key(out[j]) < key(out[i]) })
	return out
}
func SliceSortStableByKeyDescInPlace[T any, K cmp.Ordered](in []T, key func(T) K) {
	sort.SliceStable(in, func(i, j int) bool { return key(in[j]) < key(in[i]) })
}
