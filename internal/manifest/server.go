package manifest

import (
	"strings"
	"time"

	"github.com/ju4n97/hclapi/internal/problem"
	"github.com/ju4n97/hclapi/internal/scalar"
)

// ProblemConfig holds global RFC 9457 Problem Details configuration.
type ProblemConfig struct {
	TypePrefix string
}

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
	Problem      ProblemConfig
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
		Problem:      ProblemConfig{},
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
	if s.Problem.TypePrefix == "" {
		s.Problem.TypePrefix = def.Problem.TypePrefix
	}
	if s.OpenAPI.Title == "" {
		s.OpenAPI.Title = def.OpenAPI.Title
	}
	if s.OpenAPI.Version == "" {
		s.OpenAPI.Version = def.OpenAPI.Version
	}
	return s
}

// ProblemType returns the error URI using Problem.TypePrefix or the default URN prefix.
func (s Server) ProblemType(slug string) string {
	if s.Problem.TypePrefix != "" {
		prefix := s.Problem.TypePrefix
		// If it's an HTTP URL, ensure clean slash join
		if strings.HasPrefix(prefix, "http://") || strings.HasPrefix(prefix, "https://") {
			return strings.TrimSuffix(prefix, "/") + "/" + slug
		}
		// If it's a URN, append directly
		return prefix + slug
	}
	return problem.TypeURI(slug)
}
