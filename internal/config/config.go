package config

import (
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"

	"github.com/ju4n97/hclapi/internal/scalar"
)

// Config represents the complete, validated AST of an API definition.
type Config struct {
	Server      Server
	Problem     Problem
	OpenAPI     OpenAPI
	Connections []Connection
	Schemas     map[string]Schema
	Endpoints   []Endpoint
}

// Server holds transport listener settings.
type Server struct {
	Host         string
	Port         int
	ReadTimeout  scalar.Duration
	WriteTimeout scalar.Duration
	IdleTimeout  scalar.Duration
	MaxBodySize  scalar.ByteSize
}

func (s *Server) SetDefaults() {
	if s.Host == "" {
		s.Host = "127.0.0.1"
	}
	if s.Port == 0 {
		s.Port = 8080
	}
	if s.ReadTimeout == 0 {
		s.ReadTimeout = scalar.Duration(15 * time.Second)
	}
	if s.WriteTimeout == 0 {
		s.WriteTimeout = scalar.Duration(15 * time.Second)
	}
	if s.IdleTimeout == 0 {
		s.IdleTimeout = scalar.Duration(60 * time.Second)
	}
	if s.MaxBodySize == 0 {
		s.MaxBodySize = scalar.ByteSize(10 * 1024 * 1024)
	}
}

// Problem configures RFC 9457 Problem Details error type resolution.
type Problem struct {
	TypePrefix string `hcl:"type_prefix,optional"`
}

func (p Problem) FormatType(slug string) string {
	if p.TypePrefix == "" {
		return "urn:hclapi:error:" + slug
	}
	if strings.HasPrefix(p.TypePrefix, "http://") || strings.HasPrefix(p.TypePrefix, "https://") {
		return strings.TrimSuffix(p.TypePrefix, "/") + "/" + slug
	}
	return p.TypePrefix + slug
}

// OpenAPI holds global OpenAPI 3.1 specification header metadata.
type OpenAPI struct {
	Title       string         `hcl:"title,optional"`
	Version     string         `hcl:"version,optional"`
	Description string         `hcl:"description,optional"`
	ServersExpr hcl.Expression `hcl:"servers,optional"`
	TagsExpr    hcl.Expression `hcl:"tags,optional"`
	Contact     *Contact       `hcl:"contact,block"`
	License     *License       `hcl:"license,block"`

	// Evaluated metadata
	Servers []OpenAPIServer
	Tags    []OpenAPITag
}

func (o *OpenAPI) SetDefaults() {
	if o.Title == "" {
		o.Title = "API Documentation"
	}
	if o.Version == "" {
		o.Version = "1.0.0"
	}
}

type OpenAPIServer struct {
	URL         string
	Description string
}

type OpenAPITag struct {
	Name        string
	Description string
}

type Contact struct {
	Name  string `hcl:"name,optional"`
	Email string `hcl:"email,optional"`
	URL   string `hcl:"url,optional"`
}

type License struct {
	Name string `hcl:"name,optional"`
	URL  string `hcl:"url,optional"`
}

// Connection represents a database or cache connection pool.
type Connection struct {
	Driver string
	Name   string
	Source string
	Pool   PoolTune
}

type PoolTune struct {
	MaxOpen     int
	MaxIdle     int
	MaxLifetime scalar.Duration
	IdleTimeout scalar.Duration
}

func (c *Connection) SetDefaults() {
	if c.Pool.MaxOpen == 0 {
		c.Pool.MaxOpen = 25
	}
	if c.Pool.MaxIdle == 0 {
		c.Pool.MaxIdle = 5
	}
	if c.Pool.MaxLifetime == 0 {
		c.Pool.MaxLifetime = scalar.Duration(30 * time.Minute)
	}
	if c.Pool.IdleTimeout == 0 {
		c.Pool.IdleTimeout = scalar.Duration(5 * time.Minute)
	}
}

func (c Connection) Key() string {
	return c.Driver + "." + c.Name
}

func (c Connection) Reference() string {
	return "connection." + c.Driver + "." + c.Name
}

// Schema represents a reusable request validation schema.
type Schema struct {
	Name        string  `hcl:"name,label"`
	Description string  `hcl:"description,optional"`
	Fields      []Field `hcl:"field,block"`
}

// Field defines structural and semantic validation constraints on a parameter or payload.
type Field struct {
	Name        string         `hcl:"name,label"`
	TypeExpr    hcl.Expression `hcl:"type,attr"`
	Required    bool           `hcl:"required,optional"`
	DefaultExpr hcl.Expression `hcl:"default,optional"`
	Description string         `hcl:"description,optional"`
	EnumExpr    hcl.Expression `hcl:"enum,optional"`
	Format      string         `hcl:"format,optional"`
	Pattern     string         `hcl:"pattern,optional"`
	MinLength   *int           `hcl:"min_length,optional"`
	MaxLength   *int           `hcl:"max_length,optional"`
	Min         *float64       `hcl:"min,optional"`
	Max         *float64       `hcl:"max,optional"`
	MinItems    *int           `hcl:"min_items,optional"`
	MaxItems    *int           `hcl:"max_items,optional"`
	UniqueItems bool           `hcl:"unique_items,optional"`

	// Statically evaluated values:
	Type    string
	Default any
	Enum    []any
}
