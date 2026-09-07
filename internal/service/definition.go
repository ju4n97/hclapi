package service

import (
	"strings"
	"time"
)

// Definition represents the verified, immutable runtime specification for an API service.
type Definition struct {
	Server      Server
	OpenAPI     OpenAPI
	Problem     Problem
	Connections []Connection
	Schemas     map[string]Schema
	Endpoints   []Endpoint
}

// Server defines transport-level listener and socket settings.
type Server struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	MaxBodySize  int64
}

// Problem configures RFC 9457 Problem Details error type URI formatting.
type Problem struct {
	TypePrefix string
}

// FormatType returns the fully qualified error type URI for a given slug.
func (p Problem) FormatType(slug string) string {
	if p.TypePrefix == "" {
		return "urn:hclapi:error:" + slug
	}
	if strings.HasPrefix(p.TypePrefix, "http://") || strings.HasPrefix(p.TypePrefix, "https://") {
		return strings.TrimSuffix(p.TypePrefix, "/") + "/" + slug
	}
	return p.TypePrefix + slug
}

// OpenAPI holds document metadata for OpenAPI 3.1 specification generation.
type OpenAPI struct {
	Title       string
	Version     string
	Description string
	Servers     []OpenAPIServer
	Tags        []OpenAPITag
	Contact     *Contact
	License     *License
}

// OpenAPIServer defines a target deployment host in the specification.
type OpenAPIServer struct {
	URL         string
	Description string
}

// OpenAPITag defines an operation category tag.
type OpenAPITag struct {
	Name        string
	Description string
}

// Contact captures API maintainer contact information.
type Contact struct {
	Name  string
	Email string
	URL   string
}

// License captures API licensing information.
type License struct {
	Name string
	URL  string
}

// Connection represents a relational database or connection pool declaration.
type Connection struct {
	Driver string
	Name   string
	Source string
	Pool   PoolConfig
}

// Key returns the short identifier for the pool (e.g., "postgres.primary").
func (c Connection) Key() string {
	return c.Driver + "." + c.Name
}

// Reference returns the full manifest reference path (e.g., "connection.postgres.primary").
func (c Connection) Reference() string {
	return "connection." + c.Driver + "." + c.Name
}

// PoolConfig defines sizing and lifecycle limits for a connection pool.
type PoolConfig struct {
	MaxOpen     int
	MaxIdle     int
	MaxLifetime time.Duration
	IdleTimeout time.Duration
}

// Schema represents a reusable request payload schema.
type Schema struct {
	Name        string
	Description string
	Fields      []Field
}

// Field defines validation rules and bounds for an input property.
type Field struct {
	Name        string
	Type        string
	Required    bool
	Default     any
	Description string
	Enum        []any
	Format      string
	Pattern     string
	MinLength   *int
	MaxLength   *int
	Min         *float64
	Max         *float64
	MinItems    *int
	MaxItems    *int
	UniqueItems bool
}

// Endpoint represents an HTTP route with ingress rules and a terminal handler.
type Endpoint struct {
	Method       string
	Path         string
	RoutePattern string
	Description  string
	RequestRules RequestRules
	Handler      Handler
}

// RequestRules holds verified field validation constraints for an endpoint.
type RequestRules struct {
	PathFields   []Field
	QueryFields  []Field
	HeaderFields []Field
	BodyFields   []Field
}

// Handler is a sealed interface guaranteed to be either PipelineHandler or OpenAPIHandler.
type Handler interface {
	isHandler()
}

// PipelineHandler represents an endpoint executed through sequential pipeline steps.
type PipelineHandler struct {
	Steps []Step
}

func (PipelineHandler) isHandler() {}

// OpenAPIHandler represents an endpoint serving specifications or documentation UIs.
type OpenAPIHandler struct {
	Mode        string
	Format      string
	Renderer    string
	SpecURL     string
	Template    string
	Title       string
	Version     string
	Description string
}

func (OpenAPIHandler) isHandler() {}
