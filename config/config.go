package config

// Config represents the configuration file for SQLMux (sqlmux.yaml).
type Config struct {
	Server   Server   `yaml:"server"`
	Database Database `yaml:"database"`
	API      API      `yaml:"api"`
}

// Server represents the configuration for the HTTP server.
type Server struct {
	Host         string   `yaml:"host"`
	Port         int      `yaml:"port"`
	ReadTimeout  Duration `yaml:"read_timeout"`
	WriteTimeout Duration `yaml:"write_timeout"`
	IdleTimeout  Duration `yaml:"idle_timeout"`
}

// Database represents the configuration for the database connection.
type Database struct {
	Driver string `yaml:"driver"`
	URL    string `yaml:"url"`
	Pool   Pool   `yaml:"pool"`
}

// Pool represents the configuration for the database connection pool.
type Pool struct {
	MaxOpenConns    int      `yaml:"max_open_conns"`
	MaxIdleConns    int      `yaml:"max_idle_conns"`
	ConnMaxLifetime Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime Duration `yaml:"conn_max_idle_time"`
}

// API represents the configuration for the API endpoints.
type API struct {
	Prefix    string     `yaml:"prefix"`
	Resources []Resource `yaml:"resources"`
}

// Resource represents the configuration for a single API resource.
type Resource struct {
	Name     string  `yaml:"name"`
	BasePath string  `yaml:"base_path"`
	Routes   []Route `yaml:"routes"`
}

// Route represents the configuration for a single API route.
type Route struct {
	Method     string           `yaml:"method"`
	Path       string           `yaml:"path"`
	PathParams map[string]Field `yaml:"path_params"`
	Body       map[string]Field `yaml:"body"`
	Query      Query            `yaml:"query"`
	Returns    string           `yaml:"returns"`
}

// Field represents the configuration for a single API field.
type Field struct {
	Type     string `yaml:"type"`
	Required bool   `yaml:"required"`
}

// Query represents the configuration for a single API query.
type Query struct {
	SQL       string            `yaml:"sql"`
	Procedure string            `yaml:"procedure"`
	Params    map[string]string `yaml:"params"`
}
