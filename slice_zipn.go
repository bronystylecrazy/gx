package gx

import "errors"

// SliceZipN zips N columns into rows (transpose).
// Given cols = [[c0_0,c0_1,...],[c1_0,c1_1,...],...]
// returns rows = [[c0_0,c1_0,...],[c0_1,c1_1,...],...], truncated to the shortest column.
func SliceZipN[T any](cols [][]T) [][]T {
	if len(cols) == 0 {
		return nil
	}
	// find min column length
	minLen := len(cols[0])
	for i := 1; i < len(cols); i++ {
		if l := len(cols[i]); l < minLen {
			minLen = l
		}
	}
	if minLen == 0 {
		return nil
	}
	out := make([][]T, minLen)
	for r := 0; r < minLen; r++ {
		row := make([]T, len(cols))
		for c := range cols {
			row[c] = cols[c][r]
		}
		out[r] = row
	}
	return out
}

// SliceZipNStrict zips columns and returns an error if any column length differs.
func SliceZipNStrict[T any](cols [][]T) ([][]T, error) {
	if len(cols) == 0 {
		return nil, nil
	}
	want := len(cols[0])
	for i := 1; i < len(cols); i++ {
		if len(cols[i]) != want {
			return nil, errors.New("gx.SliceZipNStrict: length mismatch across columns")
		}
	}
	out := make([][]T, want)
	for r := range want {
		row := make([]T, len(cols))
		for c := range cols {
			row[c] = cols[c][r]
		}
		out[r] = row
	}
	return out, nil
}

// SliceUnzipN "unzips" rows back into columns (transpose rows→columns),
// truncating to the shortest row length.
func SliceUnzipN[T any](rows [][]T) [][]T {
	if len(rows) == 0 {
		return nil
	}
	// min width across rows
	minW := len(rows[0])
	for i := 1; i < len(rows); i++ {
		if w := len(rows[i]); w < minW {
			minW = w
		}
	}
	if minW == 0 {
		return nil
	}
	out := make([][]T, minW) // columns
	for c := 0; c < minW; c++ {
		col := make([]T, len(rows)) // column height = number of rows
		for r := range rows {
			col[r] = rows[r][c]
		}
		out[c] = col
	}
	return out
}

// SliceUnzipNStrict unzips rows back into columns and errors if any row length differs.
func SliceUnzipNStrict[T any](rows [][]T) ([][]T, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	wantW := len(rows[0])
	for i := 1; i < len(rows); i++ {
		if len(rows[i]) != wantW {
			return nil, errors.New("gx.SliceUnzipNStrict: length mismatch across rows")
		}
	}
	out := make([][]T, wantW) // columns
	for c := range wantW {
		col := make([]T, len(rows))
		for r := range rows {
			col[r] = rows[r][c]
		}
		out[c] = col
	}
	return out, nil
}

// Convenience aliases if you prefer matrix semantics:
//
// SliceTranspose is the same as SliceUnzipN (rows -> columns, truncating).
// SliceTransposeStrict is the same as SliceUnzipNStrict (rows -> columns, strict).

func SliceTranspose[T any](rows [][]T) [][]T                { return SliceUnzipN(rows) }
func SliceTransposeStrict[T any](rows [][]T) ([][]T, error) { return SliceUnzipNStrict(rows) }
