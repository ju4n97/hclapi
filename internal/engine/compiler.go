package engine

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/ju4n97/hclapi/internal/core"
	"github.com/ju4n97/hclapi/internal/parser"
)

// CompiledRequestRules holds pre-compiled field validation constraints for an endpoint.
type CompiledRequestRules struct {
	PathFields   []core.Field
	QueryFields  []core.Field
	HeaderFields []core.Field
	BodyFields   []core.Field
}

// CompiledEndpoint represents a fully verified and compiled HTTP route.
type CompiledEndpoint struct {
	MethodAndPath string
	Steps         []parser.ParsedStep
	Rules         CompiledRequestRules
}

// CompiledService represents the entire statically compiled and verified manifest tree.
type CompiledService struct {
	Server      core.Server
	Connections []core.Connection
	Schemas     map[string][]core.Field
	Endpoints   []CompiledEndpoint
}

// Compile performs static semantic analysis on the entire AST manifest tree.
// It verifies all connection and schema references and returns a ready-to-run CompiledService.
func Compile(manifest *parser.Manifest, evalCtx *hcl.EvalContext) (*CompiledService, error) {
	if manifest == nil {
		return &CompiledService{
			Server:  core.DefaultServer(),
			Schemas: make(map[string][]core.Field),
		}, nil
	}

	// Compile and validate server settings
	serverConfig, err := manifest.Server.ToServer()
	if err != nil {
		return nil, fmt.Errorf("server config: %w", err)
	}

	// Compile and validate connection blocks
	connections, connIndex, err := compileConnections(manifest.Connections)
	if err != nil {
		return nil, err
	}

	// Compile and validate schema blocks
	schemasMap, err := compileSchemas(manifest.Schemas, evalCtx)
	if err != nil {
		return nil, err
	}

	// Compile and validate endpoints and pipeline steps
	endpoints, err := compileEndpoints(manifest.Endpoints, connIndex, schemasMap, evalCtx)
	if err != nil {
		return nil, err
	}

	return &CompiledService{
		Server:      serverConfig,
		Connections: connections,
		Schemas:     schemasMap,
		Endpoints:   endpoints,
	}, nil
}

func compileConnections(blocks []parser.ConnectionBlock) ([]core.Connection, map[string]bool, error) {
	var connections []core.Connection
	connIndex := make(map[string]bool, len(blocks))

	for _, block := range blocks {
		conn, err := block.ToConnection()
		if err != nil {
			return nil, nil, fmt.Errorf("connection %q: %w", block.Name, err)
		}

		key := conn.Key()
		if connIndex[key] {
			return nil, nil, fmt.Errorf("duplicate connection declaration %q", conn.Reference())
		}

		connIndex[key] = true
		connIndex[conn.Reference()] = true
		connections = append(connections, conn)
	}

	return connections, connIndex, nil
}

func compileSchemas(blocks []parser.SchemaBlock, evalCtx *hcl.EvalContext) (map[string][]core.Field, error) {
	schemasMap := make(map[string][]core.Field, len(blocks))

	for _, block := range blocks {
		if _, exists := schemasMap[block.Name]; exists {
			return nil, fmt.Errorf("duplicate schema declaration %q", "schema."+block.Name)
		}

		var fields []core.Field
		for _, fieldBlock := range block.Fields {
			cf, err := fieldBlock.ToField(evalCtx)
			if err != nil {
				return nil, fmt.Errorf("schema %q: %w", block.Name, err)
			}
			fields = append(fields, cf)
		}
		schemasMap[block.Name] = fields
	}

	return schemasMap, nil
}

func compileEndpoints(
	blocks []parser.EndpointBlock,
	connIndex map[string]bool,
	schemasMap map[string][]core.Field,
	evalCtx *hcl.EvalContext,
) ([]CompiledEndpoint, error) {
	seenRoutes := make(map[string]bool, len(blocks))
	var endpoints []CompiledEndpoint

	for _, ep := range blocks {
		if seenRoutes[ep.MethodAndPath] {
			return nil, fmt.Errorf("duplicate endpoint route %q", ep.MethodAndPath)
		}
		seenRoutes[ep.MethodAndPath] = true

		steps, err := parser.DecodePipelineSteps(&ep.Pipeline)
		if err != nil {
			return nil, fmt.Errorf("endpoint %q: %w", ep.MethodAndPath, err)
		}

		if len(steps) == 0 {
			return nil, fmt.Errorf("endpoint %q: pipeline must declare at least one step", ep.MethodAndPath)
		}

		if err := validatePipelineSteps(ep.MethodAndPath, steps, connIndex); err != nil {
			return nil, err
		}

		rules, err := compileRequestRules(ep.MethodAndPath, ep.Request, schemasMap, evalCtx)
		if err != nil {
			return nil, err
		}

		endpoints = append(endpoints, CompiledEndpoint{
			MethodAndPath: ep.MethodAndPath,
			Steps:         steps,
			Rules:         rules,
		})
	}

	return endpoints, nil
}

func validatePipelineSteps(route string, steps []parser.ParsedStep, connIndex map[string]bool) error {
	seenStepNames := make(map[string]bool, len(steps))

	for _, step := range steps {
		if step.Name != "" {
			if seenStepNames[step.Name] {
				return fmt.Errorf("endpoint %q: duplicate step name %q in pipeline", route, step.Name)
			}
			seenStepNames[step.Name] = true
		}

		// Statically verify SQL connection references exist
		if step.Type == parser.StepTypeSQL && step.SQL != nil {
			connRef, err := parser.ResolveConnectionRef(step.SQL.Connection)
			if err != nil {
				return fmt.Errorf("endpoint %q: step %q connection: %w", route, step.Name, err)
			}

			cleanRef := strings.TrimPrefix(connRef, "connection.")
			if !connIndex[cleanRef] && !connIndex[connRef] {
				return fmt.Errorf("endpoint %q: step %q: unknown connection %q", route, step.Name, connRef)
			}
		}
	}

	return nil
}

func compileRequestRules(
	route string,
	req *parser.RequestBlock,
	schemasMap map[string][]core.Field,
	evalCtx *hcl.EvalContext,
) (CompiledRequestRules, error) {
	var rules CompiledRequestRules
	if req == nil {
		return rules, nil
	}

	compileFields := func(blocks []parser.FieldBlock) ([]core.Field, error) {
		var fields []core.Field
		for _, fieldBlock := range blocks {
			cf, err := fieldBlock.ToField(evalCtx)
			if err != nil {
				return nil, err
			}
			fields = append(fields, cf)
		}
		return fields, nil
	}

	resolveTarget := func(targetName string, inline *parser.FieldGroupBlock, expr hcl.Expression) ([]core.Field, error) {
		if inline != nil {
			return compileFields(inline.Fields)
		}
		if expr != nil {
			schemaRef, err := parser.ResolveSchemaRef(expr)
			if err != nil {
				return nil, fmt.Errorf("%s schema: %w", targetName, err)
			}
			fields, exists := schemasMap[schemaRef]
			if !exists {
				return nil, fmt.Errorf("unknown schema reference %q", "schema."+schemaRef)
			}
			return fields, nil
		}
		return nil, nil
	}

	var err error
	if rules.PathFields, err = resolveTarget("path", req.PathInline, req.PathExpr); err != nil {
		return rules, fmt.Errorf("endpoint %q: %w", route, err)
	}
	if rules.QueryFields, err = resolveTarget("query", req.QueryInline, req.QueryExpr); err != nil {
		return rules, fmt.Errorf("endpoint %q: %w", route, err)
	}
	if rules.HeaderFields, err = resolveTarget("headers", req.HeadersInline, req.HeadersExpr); err != nil {
		return rules, fmt.Errorf("endpoint %q: %w", route, err)
	}
	if rules.BodyFields, err = resolveTarget("body", req.BodyInline, req.BodyExpr); err != nil {
		return rules, fmt.Errorf("endpoint %q: %w", route, err)
	}

	return rules, nil
}
