package core

import "time"

// Server defines the resolved HTTP server configuration.
type Server struct {
	Host         string
	Port         int
	ReadTimeout  Duration
	WriteTimeout Duration
	IdleTimeout  Duration
	MaxBodySize  ByteSize
}

// DefaultServer returns baseline production configuration values.
func DefaultServer() Server {
	return Server{
		Host:         "127.0.0.1",
		Port:         8080,
		ReadTimeout:  Duration(15 * time.Second),
		WriteTimeout: Duration(15 * time.Second),
		IdleTimeout:  Duration(60 * time.Second),
		MaxBodySize:  ByteSize(10 * 1024 * 1024),
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
