package manifest

import (
	"time"

	"github.com/ju4n97/hclapi/internal/scalar"
)

// Server defines the resolved HTTP server transport configuration.
type Server struct {
	Host         string
	Port         int
	ReadTimeout  scalar.Duration
	WriteTimeout scalar.Duration
	IdleTimeout  scalar.Duration
	MaxBodySize  scalar.ByteSize
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
	return s
}
