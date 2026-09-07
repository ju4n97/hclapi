package manifest

// OpenAPIServer defines a target server deployment in the OpenAPI spec.
type OpenAPIServer struct {
	URL         string
	Description string
}

// OpenAPITag defines an operation category tag.
type OpenAPITag struct {
	Name        string
	Description string
}

// OpenAPIContact defines API contact details.
type OpenAPIContact struct {
	Name  string
	Email string
	URL   string
}

// OpenAPILicense defines API licensing information.
type OpenAPILicense struct {
	Name string
	URL  string
}

// OpenAPIConfig holds global OpenAPI 3.1 document header metadata.
type OpenAPIConfig struct {
	Title       string
	Version     string
	Description string
	Servers     []OpenAPIServer
	Tags        []OpenAPITag
	Contact     *OpenAPIContact
	License     *OpenAPILicense
}

// DefaultOpenAPIConfig returns baseline production OpenAPI metadata.
func DefaultOpenAPIConfig() OpenAPIConfig {
	return OpenAPIConfig{
		Title:   "API Documentation",
		Version: "1.0.0",
	}
}

// WithDefaults returns a copy of OpenAPIConfig with baseline defaults applied.
func (o OpenAPIConfig) WithDefaults() OpenAPIConfig {
	def := DefaultOpenAPIConfig()
	if o.Title == "" {
		o.Title = def.Title
	}
	if o.Version == "" {
		o.Version = def.Version
	}
	return o
}
