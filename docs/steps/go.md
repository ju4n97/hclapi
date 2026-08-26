# Go step

The `go` step invokes custom in-process Go functions registered on the `Engine` instance.

## Syntax

```hcl
go "hash_secret" {
  use = "crypto.bcrypt_hash"
  args = {
    password = ctx.request.body.password
    cost     = 12
  }
}
```

## Host implementation

```go
engine.RegisterStep("crypto.bcrypt_hash", func(ctx *hclapi.Context) (any, error) {
	rawPassword, ok := ctx.Args["password"].(string)
	if !ok || len(rawPassword) == 0 {
		return nil, errors.New("missing password argument")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return string(hash), nil
})
```

## Panic recovery

The `xgo` runner wraps function calls in a deferred recovery handler. Runtime panics inside user-defined Go steps are converted into standard error values and returned as HTTP 500 responses without crashing the host process.
