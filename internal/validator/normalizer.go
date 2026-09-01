package validator

import (
	"fmt"
	"maps"

	"github.com/ju4n97/hclapi/internal/core"
)

// Normalize applies default values and normalizes missing optional fields to nil.
func Normalize(data map[string]any, fields []core.Field) map[string]any {
	result := make(map[string]any, len(fields)+len(data))
	maps.Copy(result, data)

	for _, field := range fields {
		val, exists := result[field.Name]

		if !exists || val == nil {
			if field.Default != nil {
				result[field.Name] = field.Default
			} else if !field.Required {
				result[field.Name] = nil // Explicit nil so HCL can traverse as null
			}
		}
	}

	return result
}

// NormalizeStringMap injects default values directly into maps without heap allocations.
func NormalizeStringMap(data map[string]string, fields []core.Field) {
	for _, field := range fields {
		val, exists := data[field.Name]
		if (!exists || val == "") && field.Default != nil {
			data[field.Name] = fmt.Sprintf("%v", field.Default)
		}
	}
}
