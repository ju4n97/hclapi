# Custom step registry

Registering in-process native Go handlers to execute within HCL pipelines.

## Step registration interface

```go
engine.RegisterStep("crypto.hash_password", func(ctx *hclapi.Context) (any, error) {
	rawPassword, ok := ctx.Args["password"].(string)
	if !ok || len(rawPassword) == 0 {
		return nil, errors.New("missing or invalid 'password' argument")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing failed: %w", err)
	}

	return string(hashedBytes), nil
})
```

## Invoking from HCL

```hcl
endpoint "POST /auth/register" {
  pipeline {
    go "hash" {
      use = "crypto.hash_password"
      args = {
        password = ctx.request.body.password
      }
    }
  }
}
```
