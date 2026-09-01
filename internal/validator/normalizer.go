package validator

import (
	"maps"
	"strconv"
	"strings"

	"github.com/ju4n97/hclapi/internal/core"
)

// Normalize injects configured defaults, normalizes omitted optional fields to nil,
// and coerces string inputs (from path and query params) to their declared types.
func Normalize(data map[string]any, fields []core.Field) map[string]any {
	result := make(map[string]any, len(data)+len(fields))
	maps.Copy(result, data)

	for _, field := range fields {
		val, exists := data[field.Name]
		if !exists || val == nil {
			if field.Default != nil {
				result[field.Name] = field.Default
			} else if field.Required {
				result[field.Name] = nil // Explicit nil so HCL can traverse as null
			}
			continue
		}

		// Coerce string representations from query/path params into declared types
		if strVal, isStr := val.(string); isStr {
			switch field.Type {
			case "int":
				if intVal, err := strconv.ParseInt(strVal, 10, 64); err == nil {
					result[field.Name] = intVal
				}
			case "float":
				if floatVal, err := strconv.ParseFloat(strVal, 64); err == nil {
					result[field.Name] = floatVal
				}
			case "bool":
				switch strings.ToLower(strings.TrimSpace(strVal)) {
				case "true", "1", "yes", "on":
					result[field.Name] = true
				case "false", "0", "no", "off":
					result[field.Name] = false
				}
			}
		}
	}

	return result
}
