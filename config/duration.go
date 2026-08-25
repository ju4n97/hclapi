package config

import (
	"time"

	"go.yaml.in/yaml/v4"
)

// Duration wraps time.Duration so it can be unmarshalled from Go-style duration
// strings (e.g. "10s", "30m", "1h") as used in the config file.
type Duration time.Duration

func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(parsed)
	return nil
}

func (d Duration) MarshalYAML() (any, error) {
	return d.Duration().String(), nil
}
