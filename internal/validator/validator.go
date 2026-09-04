// Package validator enforces OpenAPI 3.1 schema types, format constraints, and default value normalization.
package validator

import (
	"fmt"
	"maps"
	"net"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ju4n97/hclapi/internal/manifest"
	"github.com/ju4n97/hclapi/internal/problem"
	"github.com/ju4n97/hclapi/internal/scalar"
)

var (
	uuidRegex = regexp.MustCompile(
		`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`,
	)
	hostnameRegex = regexp.MustCompile(
		`^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$`,
	)
	// patternCache is a thread-safe cache to avoid recompiling regex patterns on every HTTP request.
	patternCache sync.Map
)

// ValidateBody validates and normalizes a JSON request body in a single pass.
// It applies defaults, normalizes missing optional fields to nil, and validates all constraints.
func ValidateBody(data map[string]any, fields []manifest.Field) (map[string]any, []problem.InvalidParam) {
	result := make(map[string]any, len(fields)+len(data))
	if len(data) > 0 {
		maps.Copy(result, data)
	}
	var invalidParams []problem.InvalidParam

	for _, field := range fields {
		val, exists := result[field.Name]

		if !exists || val == nil {
			switch {
			case field.Default != nil:
				result[field.Name] = field.Default
				val = field.Default
			case field.Required:
				invalidParams = append(invalidParams, problem.InvalidParam{
					Name:   field.Name,
					Reason: "field is required",
				})
				continue
			default:
				result[field.Name] = nil // Explicit nil so HCL can traverse as null
				continue
			}
		}

		if field.Required {
			if strVal, ok := val.(string); ok && strings.TrimSpace(strVal) == "" {
				invalidParams = append(invalidParams, problem.InvalidParam{
					Name:   field.Name,
					Reason: "field is required and cannot be empty",
				})
				continue
			}
		}

		if errReason := validateValue(val, field); errReason != "" {
			invalidParams = append(invalidParams, problem.InvalidParam{
				Name:   field.Name,
				Reason: errReason,
			})
		}
	}

	return result, invalidParams
}

// ValidateStringMap validates string-keyed parameter maps (Path, Query) and injects defaults.
func ValidateStringMap(data map[string]string, fields []manifest.Field) []problem.InvalidParam {
	return validateStringMapWithLookup(data, fields, func(name string) string {
		return name
	})
}

// ValidateHeaders validates incoming HTTP headers against schema fields in a single pass.
// Per RFC 9110, header lookup is case-insensitive against lowercased ingress headers,
// defaults are injected, and error diagnostics retain the author's declared schema casing.
func ValidateHeaders(headers map[string]string, fields []manifest.Field) []problem.InvalidParam {
	return validateStringMapWithLookup(headers, fields, strings.ToLower)
}

func validateStringMapWithLookup(
	data map[string]string,
	fields []manifest.Field,
	keyLookup func(string) string,
) []problem.InvalidParam {
	var invalidParams []problem.InvalidParam

	for _, field := range fields {
		lookupKey := keyLookup(field.Name)
		rawStr, exists := data[lookupKey]

		if !exists || strings.TrimSpace(rawStr) == "" {
			if field.Default != nil {
				data[lookupKey] = fmt.Sprintf("%v", field.Default)
				continue
			}
			if field.Required {
				invalidParams = append(invalidParams, problem.InvalidParam{
					Name:   field.Name,
					Reason: "field is required and cannot be empty",
				})
			}
			continue
		}

		coercedVal, errReason := coerceString(rawStr, field.Type)
		if errReason != "" {
			invalidParams = append(invalidParams, problem.InvalidParam{
				Name:   field.Name,
				Reason: errReason,
			})
			continue
		}

		if errReason := validateValue(coercedVal, field); errReason != "" {
			invalidParams = append(invalidParams, problem.InvalidParam{
				Name:   field.Name,
				Reason: errReason,
			})
		}
	}

	return invalidParams
}

func validateValue(val any, field manifest.Field) string {
	switch {
	case field.Type == "string":
		strVal, ok := val.(string)
		if !ok {
			return "must be of type string"
		}
		return checkStringConstraints(strVal, field)

	case field.Type == "int":
		intVal, ok := scalar.ToInt64(val)
		if !ok {
			return "must be of type int"
		}
		return checkNumericConstraints(float64(intVal), intVal, field)

	case field.Type == "float":
		floatVal, ok := scalar.ToFloat64(val)
		if !ok {
			return "must be of type float"
		}
		return checkNumericConstraints(floatVal, floatVal, field)

	case field.Type == "bool":
		if _, ok := val.(bool); !ok {
			return "must be of type bool"
		}

	case strings.HasPrefix(field.Type, "list"):
		return checkListConstraints(val, field)

	case strings.HasPrefix(field.Type, "map"):
		rv := reflect.ValueOf(val)
		if rv.Kind() != reflect.Map {
			return "must be of type map"
		}

	case field.Type == "any":
		return ""
	}

	return ""
}

func checkStringConstraints(val string, field manifest.Field) string {
	runes := []rune(val)
	if field.MinLength != nil && len(runes) < *field.MinLength {
		return fmt.Sprintf("length must be at least %d characters", *field.MinLength)
	}
	if field.MaxLength != nil && len(runes) > *field.MaxLength {
		return fmt.Sprintf("length must be at most %d characters", *field.MaxLength)
	}
	if field.Pattern != "" {
		if !matchPattern(field.Pattern, val) {
			return "must match pattern " + field.Pattern
		}
	}
	if field.Format != "" {
		if !validateFormat(val, field.Format) {
			return fmt.Sprintf("must be a valid %s format", field.Format)
		}
	}
	if len(field.Enum) > 0 {
		return checkEnum(val, field.Enum)
	}
	return ""
}

func matchPattern(pattern, val string) bool {
	var re *regexp.Regexp
	if cached, ok := patternCache.Load(pattern); ok {
		re = cached.(*regexp.Regexp)
	} else {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return false
		}
		patternCache.Store(pattern, compiled)
		re = compiled
	}
	return re.MatchString(val)
}

func checkNumericConstraints(val float64, rawVal any, field manifest.Field) string {
	if field.Min != nil && val < *field.Min {
		return fmt.Sprintf("must be greater than or equal to %v", *field.Min)
	}
	if field.Max != nil && val > *field.Max {
		return fmt.Sprintf("must be less than or equal to %v", *field.Max)
	}
	if len(field.Enum) > 0 {
		return checkEnum(rawVal, field.Enum)
	}
	return ""
}

func checkListConstraints(val any, field manifest.Field) string {
	rv := reflect.ValueOf(val)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return "must be of type list"
	}

	count := rv.Len()
	if field.MinItems != nil && count < *field.MinItems {
		return fmt.Sprintf("must contain at least %d items", *field.MinItems)
	}
	if field.MaxItems != nil && count > *field.MaxItems {
		return fmt.Sprintf("must contain at most %d items", *field.MaxItems)
	}
	if field.UniqueItems && hasDuplicates(rv) {
		return "items must be unique"
	}
	return ""
}

func checkEnum(val any, enumList []any) string {
	target := fmt.Sprintf("%v", val)
	for _, item := range enumList {
		if fmt.Sprintf("%v", item) == target {
			return ""
		}
	}

	var formatted []string
	for _, item := range enumList {
		formatted = append(formatted, fmt.Sprintf("%q", fmt.Sprintf("%v", item)))
	}
	return fmt.Sprintf("must be one of: [%s]", strings.Join(formatted, ", "))
}

func validateFormat(val, format string) bool {
	switch strings.ToLower(format) {
	case "email":
		_, err := mail.ParseAddress(val)
		return err == nil && strings.Contains(val, "@") && strings.Contains(val, ".")
	case "uuid":
		return uuidRegex.MatchString(val)
	case "uri":
		u, err := url.ParseRequestURI(val)
		return err == nil && u.Scheme != "" && u.Host != ""
	case "date-time":
		_, err := time.Parse(time.RFC3339, val)
		return err == nil
	case "date":
		_, err := time.Parse("2006-01-02", val)
		return err == nil
	case "ipv4":
		ip := net.ParseIP(val)
		return ip != nil && ip.To4() != nil
	case "ipv6":
		ip := net.ParseIP(val)
		return ip != nil && ip.To4() == nil
	case "hostname":
		return hostnameRegex.MatchString(val)
	default:
		return true
	}
}

func coerceString(raw, targetType string) (any, string) {
	trimmed := strings.TrimSpace(raw)
	switch targetType {
	case "string":
		return raw, ""
	case "int":
		if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return i, ""
		}
		return nil, "must be of type int"
	case "float":
		if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return f, ""
		}
		return nil, "must be of type float"
	case "bool":
		switch strings.ToLower(trimmed) {
		case "true", "1", "yes", "on":
			return true, ""
		case "false", "0", "no", "off":
			return false, ""
		default:
			return nil, "must be of type bool"
		}
	default:
		return raw, ""
	}
}

func hasDuplicates(rv reflect.Value) bool {
	seen := make(map[string]bool, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		key := fmt.Sprintf("%v", rv.Index(i).Interface())
		if seen[key] {
			return true
		}
		seen[key] = true
	}
	return false
}
