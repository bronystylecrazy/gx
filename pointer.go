package gx

import "github.com/google/uuid"

func Ptr[T any](v T) *T {
	return &v
}

// Uint8Ptr converts a UUID string or *string to *[]uint8, with an optional default value.
func Uint8Ptr[T string | *string](id T) *[]uint8 {
	var strID string

	// Handle input types (string or *string)
	switch v := any(id).(type) {
	case string:
		strID = v
	case *string:
		if v == nil {
			return nil // Safe handling for nil pointer
		}
		strID = *v
	default:
		return nil // Invalid type
	}

	uuidBytes, err := uuid.Parse(strID)
	if err != nil {
		return nil // Invalid UUID
	}
	bytes := uuidBytes[:]
	return &bytes
}
