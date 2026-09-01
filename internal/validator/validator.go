// Package validator enforces OpenAPI 3.1 schema types, format constraints, and default value normalization.
package validator

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ju4n97/hclapi/internal/core"
)

var (
	uuidRegex = regexp.MustCompile(
		`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`,
	)
	hostnameRegex = regexp.MustCompile(
		`^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$`,
	)
)

// Validate checks a JSON-decoded map against compiled schema field rules.
func Validate(data map[string]any, fields []core.Field) []core.InvalidParam {
	var invalidParams []core.InvalidParam

	for _, field := range fields {
		val, exists := data[field.Name]

		if !exists || val == nil {
			if field.Required {
				invalidParams = append(invalidParams, core.InvalidParam{
					Name:   field.Name,
					Reason: "field is required",
				})
			}
			continue
		}

		if errReason := validateValue(val, field); errReason != "" {
			invalidParams = append(invalidParams, core.InvalidParam{
				Name:   field.Name,
				Reason: errReason,
			})
		}
	}

	return invalidParams
}

// ValidateStringMap validates string-keyed parameter maps (query, path, headers) with zero map allocations.
func ValidateStringMap(data map[string]string, fields []core.Field) []core.InvalidParam {
	var invalidParams []core.InvalidParam

	for _, field := range fields {
		rawStr, exists := data[field.Name]

		if !exists || rawStr == "" {
			if field.Required {
				invalidParams = append(invalidParams, core.InvalidParam{
					Name:   field.Name,
					Reason: "field is required",
				})
			}
			continue
		}

		// Coerce string to target type for constraint checking
		coercedVal, errReason := coerceString(rawStr, field.Type)
		if errReason != "" {
			invalidParams = append(invalidParams, core.InvalidParam{
				Name:   field.Name,
				Reason: errReason,
			})
			continue
		}

		if errReason := validateValue(coercedVal, field); errReason != "" {
			invalidParams = append(invalidParams, core.InvalidParam{
				Name:   field.Name,
				Reason: errReason,
			})
		}
	}

	return invalidParams
}

func validateValue(val any, field core.Field) string {
	switch {
	case field.Type == "string":
		strVal, ok := val.(string)
		if !ok {
			return "must be of type string"
		}
		return checkStringConstraints(strVal, field)

	case field.Type == "int":
		intVal, ok := toInt64(val)
		if !ok {
			return "must be of type int"
		}
		return checkNumericConstraints(float64(intVal), intVal, field)

	case field.Type == "float":
		floatVal, ok := toFloat64(val)
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

func checkStringConstraints(val string, field core.Field) string {
	runes := []rune(val)
	if field.MinLength != nil && len(runes) < *field.MinLength {
		return fmt.Sprintf("length must be at least %d characters", *field.MinLength)
	}
	if field.MaxLength != nil && len(runes) > *field.MaxLength {
		return fmt.Sprintf("length must be at most %d characters", *field.MaxLength)
	}
	if field.Pattern != "" {
		matched, err := regexp.MatchString(field.Pattern, val)
		if err != nil || !matched {
			return fmt.Sprintf("must match pattern %s", field.Pattern)
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

func checkNumericConstraints(val float64, rawVal any, field core.Field) string {
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

func checkListConstraints(val any, field core.Field) string {
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

func toInt64(val any) (int64, bool) {
	switch v := val.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		if v == float64(int64(v)) {
			return int64(v), true
		}
	}
	return 0, false
}

func toFloat64(val any) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}
