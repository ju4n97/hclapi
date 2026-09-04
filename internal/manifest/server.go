package manifest

import (
	"time"

	"github.com/ju4n97/hclapi/internal/problem"
	"github.com/ju4n97/hclapi/internal/scalar"
)

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

// Server defines the resolved HTTP server configuration.
type Server struct {
	Host         string
	Port         int
	ReadTimeout  scalar.Duration
	WriteTimeout scalar.Duration
	IdleTimeout  scalar.Duration
	MaxBodySize  scalar.ByteSize
	ErrorBaseURL string // TODO: validate introducing a new block to put stuff like this
	OpenAPI      OpenAPIConfig
}

// DefaultServer returns baseline production configuration values.
func DefaultServer() Server {
	return Server{
		Host:         "127.0.0.1",
		Port:         8080,
		ReadTimeout:  scalar.Duration(15 * time.Second),
		WriteTimeout: scalar.Duration(15 * time.Second),
		IdleTimeout:  scalar.Duration(60 * time.Second),
		MaxBodySize:  scalar.ByteSize(10 * 1024 * 1024),
		OpenAPI: OpenAPIConfig{
			Title:   "API Documentation",
			Version: "1.0.0",
		},
	}
}

// WithDefaults returns a copy of Server with any zero values replaced by baseline defaults.
func (s Server) WithDefaults() Server {
	def := DefaultServer()
	if s.Host == "" {
		s.Host = def.Host
	}
	if s.Port == 0 {
		s.Port = def.Port
	}
	if s.ReadTimeout == 0 {
		s.ReadTimeout = def.ReadTimeout
	}
	if s.WriteTimeout == 0 {
		s.WriteTimeout = def.WriteTimeout
	}
	if s.IdleTimeout == 0 {
		s.IdleTimeout = def.IdleTimeout
	}
	if s.MaxBodySize == 0 {
		s.MaxBodySize = def.MaxBodySize
	}
	if s.OpenAPI.Title == "" {
		s.OpenAPI.Title = def.OpenAPI.Title
	}
	if s.OpenAPI.Version == "" {
		s.OpenAPI.Version = def.OpenAPI.Version
	}
	return s
}

// ProblemType returns the error URI using ErrorBaseURL or the default URN prefix.
func (s Server) ProblemType(slug string) string {
	if s.ErrorBaseURL != "" {
		return s.ErrorBaseURL + "/" + slug
	}
	return problem.TypeURI(slug)
}
