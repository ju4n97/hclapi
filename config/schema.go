package config

import (
	"bytes"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"

	_ "embed"
)

//go:embed schema.json
var schemaJSON string

var configSchema *jsonschema.Schema

func init() {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader([]byte(schemaJSON)))
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal schema: %w", err))
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", doc); err != nil {
		panic(fmt.Errorf("failed to add schema: %w", err))
	}

	configSchema, err = compiler.Compile("schema.json")
	if err != nil {
		panic(fmt.Errorf("failed to compile schema: %w", err))
	}
}

func validate(instance any) error {
	if err := configSchema.Validate(instance); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return nil
}
