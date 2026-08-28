# go

Invokes a native Go function registered on the engine. Use this step for
logic that cannot be expressed in Starlark: proprietary business rules,
third-party SDKs, outbound HTTP clients, or hardware cryptography.

## Declaration

```hcl
go "<name>" {
  use  = "<registered_function_name>"
  args = {
    city = ctx.request.path.city
  }
}
```

## Attributes

| Attribute | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| label | `string` | yes | step identifier |
| `use` | `string` | yes | function name registered on the `Engine` |
| `args` | `map` | no | evaluated and passed into `ctx.Args` |

## Registration

```go
engine.RegisterStep("services.weather_lookup", func(ctx *hclapi.Context) (any, error) {
  city, ok := ctx.Args["city"].(string)
  if !ok || len(city) == 0 {
    return nil, errors.New("missing or invalid 'city' argument")
  }

  return map[string]any{
    "city":          strings.ToLower(strings.TrimSpace(city)),
    "temperature_c": 22.5,
    "condition":     "Sunny",
  }, nil
})
```

The return value is written to `steps.<name>.result`. See
[Registering steps](../go/registering-steps.md) for the full API.

## Panic recovery

A panic in a registered function is recovered by the engine, converted to
an error, and returned as a 500. The server process is unaffected.
