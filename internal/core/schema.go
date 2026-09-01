package core

// Field represents a compiled, type-safe schema field rule.
type Field struct {
	Name        string
	Type        string // "string", "int", "float", "bool", "any", "list(string)", etc.
	Required    bool
	Default     any // Evaluated static default or nil
	Description string
	Enum        []any  // Evaluated allowed values or nil
	Format      string // "email", "uuid", "date-time", etc.
	Pattern     string // Regex pattern or empty
	MinLength   *int
	MaxLength   *int
	Min         *float64
	Max         *float64
	MinItems    *int
	MaxItems    *int
	UniqueItems bool
}

// Schema represents a compiled, named validation schema.
type Schema struct {
	Name        string
	Description string
	Fields      []Field
}
