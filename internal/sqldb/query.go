package sqldb

import (
	"regexp"
)

var namedParamRegex = regexp.MustCompile(`@([a-zA-Z0-9_]+)`)

// RewriteNamedQuery converts @name parameter placeholders into dialect placeholders ($1, ?, @p1)
// and returns the ordered slice of arguments matching the placeholders.
func RewriteNamedQuery(query string, args map[string]any, d Dialect) (string, []any, error) {
	var orderedArgs []any
	var count int

	rewritten := namedParamRegex.ReplaceAllStringFunc(query, func(match string) string {
		paramName := match[1:] // strip '@'
		var val any
		if args != nil {
			val = args[paramName]
		}
		orderedArgs = append(orderedArgs, val)
		placeholder := d.Placeholder(count, paramName)
		count++
		return placeholder
	})

	return rewritten, orderedArgs, nil
}
