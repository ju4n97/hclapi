package validator

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
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

// Validate validates an input data map against compiled domain fields.
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

		if errReason := validateTypeAndConstraints(val, field); errReason != "" {
			invalidParams = append(invalidParams, core.InvalidParam{
				Name:   field.Name,
				Reason: errReason,
			})
		}
	}

	return invalidParams
}

func validateTypeAndConstraints(val any, field core.Field) string {
	switch {
	case field.Type == "string":
		strVal, ok := val.(string)
		if !ok {
			return "must be of type string"
		}

		if field.MinLength != nil && len([]rune(strVal)) < *field.MinLength {
			return fmt.Sprintf("length must be at least %d characters", *field.MinLength)
		}
		if field.MaxLength != nil && len([]rune(strVal)) > *field.MaxLength {
			return fmt.Sprintf("length must be at most %d characters", *field.MaxLength)
		}
		if field.Pattern != "" {
			matched, err := regexp.MatchString(field.Pattern, strVal)
			if err != nil || !matched {
				return fmt.Sprintf("must match pattern %s", field.Pattern)
			}
		}
		if field.Format != "" {
			if !validateFormat(strVal, field.Format) {
				return fmt.Sprintf("must be a valid %s format", field.Format)
			}
		}
		if reason := validateEnum(strVal, field.Enum); reason != "" {
			return reason
		}

	case field.Type == "int":
		numVal, ok := toInt64(val)
		if !ok {
			return "must be of type int"
		}

		if field.Min != nil && float64(numVal) < *field.Min {
			return fmt.Sprintf("must be greater than or equal to %v", *field.Min)
		}
		if field.Max != nil && float64(numVal) > *field.Max {
			return fmt.Sprintf("must be less than or equal to %v", *field.Max)
		}
		if reason := validateEnum(numVal, field.Enum); reason != "" {
			return reason
		}

	case field.Type == "float":
		fVal, ok := toFloat64(val)
		if !ok {
			return "must be of type float"
		}

		if field.Min != nil && fVal < *field.Min {
			return fmt.Sprintf("must be greater than or equal to %v", *field.Min)
		}
		if field.Max != nil && fVal > *field.Max {
			return fmt.Sprintf("must be less than or equal to %v", *field.Max)
		}
		if reason := validateEnum(fVal, field.Enum); reason != "" {
			return reason
		}

	case field.Type == "bool":
		if _, ok := val.(bool); !ok {
			return "must be of type bool"
		}

	case strings.HasPrefix(field.Type, "list"):
		rv := reflect.ValueOf(val)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return "must be of type list"
		}

		itemCount := rv.Len()
		if field.MinItems != nil && itemCount < *field.MinItems {
			return fmt.Sprintf("must contain at least %d items", *field.MinItems)
		}
		if field.MaxItems != nil && itemCount > *field.MaxItems {
			return fmt.Sprintf("must contain at most %d items", *field.MaxItems)
		}
		if field.UniqueItems && hasDuplicates(rv) {
			return "items must be unique"
		}

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

func validateEnum(val any, enumList []any) string {
	if len(enumList) == 0 {
		return ""
	}

	for _, item := range enumList {
		if fmt.Sprintf("%v", item) == fmt.Sprintf("%v", val) {
			return ""
		}
	}

	var items []string
	for _, item := range enumList {
		items = append(items, fmt.Sprintf("%q", fmt.Sprintf("%v", item)))
	}
	return fmt.Sprintf("must be one of: [%s]", strings.Join(items, ", "))
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
