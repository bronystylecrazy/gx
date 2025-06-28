package gx

import (
	"github.com/google/uuid"
)

var NilUUID = uuid.Nil

func UUID[T string | *string | []uint8 | *[]uint8](id T) uuid.UUID {
	var strID string

	// Handle input types (string or *string)
	switch v := any(id).(type) {
	case string:
		strID = v
	case *string:
		if v == nil {
			return NilUUID // Safe handling for nil pointer
		}
		strID = *v
	case []uint8:
		strID = string(v)
	case *[]uint8:
		if v == nil {
			return NilUUID // Safe handling for nil pointer
		}
		strID = string(*v)
	default:
		return NilUUID // Invalid type
	}

	uuid, err := uuid.Parse(strID)
	if err != nil {
		return NilUUID // Invalid UUID
	}

	return uuid
}

// UUIDPtr converts a UUID string or *string to *uuid.UUID, with an optional default value.
func UUIDPtr[T string | *string | []uint8 | *[]uint8](id T) *uuid.UUID {
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
	case []uint8:
		strID = string(v)
	case *[]uint8:
		if v == nil {
			return nil // Safe handling for nil pointer
		}
		strID = string(*v)
	default:
		return nil // Invalid type
	}

	uuid, err := uuid.Parse(strID)
	if err != nil {
		return nil // Invalid UUID
	}

	return &uuid
}
