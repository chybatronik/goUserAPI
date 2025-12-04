package validation

import (
	"fmt"
	"sync"
)

// String buffer pool for memory-efficient string operations
var stringBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 1024)
	},
}

// ValidateFieldMemorySafe validates a field with memory-safe operations
func ValidateFieldMemorySafe(field, fieldName string, maxLen int) error {
	// Get buffer from pool
	buf := stringBufferPool.Get().([]byte)
	defer stringBufferPool.Put(buf[:0])

	// Bounded string processing - check length first
	if len(field) > maxLen {
		return fmt.Errorf("field %s exceeds maximum length of %d characters", fieldName, maxLen)
	}

	// Process in chunks to prevent memory exhaustion
	chunkSize := 256
	for i := 0; i < len(field); i += chunkSize {
		end := i + chunkSize
		if end > len(field) {
			end = len(field)
		}

		chunk := field[i:end]
		if err := ValidateUnicodeSecurity(chunk); err != nil {
			return fmt.Errorf("unicode security validation failed for field %s: %w", fieldName, err)
		}
	}

	return nil
}

// ValidatePayloadSize validates the size of incoming payloads
func ValidatePayloadSize(payload []byte, maxSize int64) error {
	if payload == nil {
		return nil
	}

	if int64(len(payload)) > maxSize {
		return fmt.Errorf("payload size %d exceeds maximum allowed size of %d bytes", len(payload), maxSize)
	}

	return nil
}

// ValidateMultipleFields validates multiple fields with memory safety
func ValidateMultipleFields(fields map[string]string, maxLen int) []error {
	var errors []error

	for fieldName, fieldValue := range fields {
		if err := ValidateFieldMemorySafe(fieldValue, fieldName, maxLen); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// ValidateInputBatch validates a batch of inputs efficiently
type InputField struct {
	Name   string
	Value  string
	MaxLen int
}

func ValidateInputBatch(inputs []InputField) []error {
	var errors []error

	for _, input := range inputs {
		if err := ValidateFieldMemorySafe(input.Value, input.Name, input.MaxLen); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// SafeStringFromBytes safely converts bytes to string with length limits
func SafeStringFromBytes(data []byte, maxLen int) string {
	if data == nil {
		return ""
	}

	if len(data) > maxLen {
		return string(data[:maxLen])
	}

	return string(data)
}

// TruncateString safely truncates string to maximum length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// ValidateStringLength validates string length within min and max bounds
func ValidateStringLength(s string, minLen, maxLen int) error {
	if len(s) < minLen {
		return fmt.Errorf("string length %d is less than minimum %d", len(s), minLen)
	}
	if len(s) > maxLen {
		return fmt.Errorf("string length %d exceeds maximum %d", len(s), maxLen)
	}
	return nil
}
