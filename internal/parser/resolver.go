package parser

import (
	"errors"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// ResolveConnectionRef extracts the connection identifier string from an HCL expression.
// It handles unquoted traversals (e.g. connection.postgres.main) and string literals (e.g. connection: "postgres.main").
func ResolveConnectionRef(expr hcl.Expression) (string, error) {
	if expr == nil {
		return "", errors.New("missing connection reference expression")
	}

	// If it's a traversal
	vars := expr.Variables()
	if len(vars) > 0 {
		var parts []string
		for _, split := range vars[0] {
			switch step := split.(type) {
			case hcl.TraverseRoot:
				parts = append(parts, step.Name)
			case hcl.TraverseAttr:
				parts = append(parts, step.Name)
			}
		}
		return strings.Join(parts, "."), nil
	}

	// If it's a string literal or evaluated expression
	val, diags := expr.Value(nil)
	if !diags.HasErrors() && val.Type().Equals(cty.String) {
		return val.AsString(), nil
	}

	return "", errors.New("invalid connection reference expression")
}

// ResolveSchemaRef extracts the schema identifier string from an HCL expression (e.g. schema.user_create -> "user_create").
func ResolveSchemaRef(expr hcl.Expression) (string, error) {
	if expr == nil {
		return "", errors.New("missing schema reference expression")
	}

	vars := expr.Variables()
	if len(vars) > 0 {
		for _, split := range vars[0] {
			if attr, ok := split.(hcl.TraverseAttr); ok {
				return attr.Name, nil
			}
		}

		if root, ok := vars[0][0].(hcl.TraverseRoot); ok && len(vars[0]) == 1 {
			return root.Name, nil
		}
	}

	val, diags := expr.Value(nil)
	if !diags.HasErrors() && val.IsKnown() && !val.IsNull() && val.Type().Equals(cty.String) {
		return strings.TrimPrefix(val.AsString(), "schema."), nil
	}

	return "", errors.New("invalid schema reference expression")
}
