# Installation

hclapi is distributed as a single, cross-platform binary.

## Precompiled binaries

Download a binary for your platform from the
[releases page](https://github.com/ju4n97/hclapi/releases).

```sh
curl -sSL https://github.com/ju4n97/hclapi/releases/latest/download/hclapi_linux_amd64.tar.gz | tar -xz
sudo mv hclapi /usr/local/bin/
```

## Build from source

Requires the [Go toolchain](https://go.dev/doc/install).

```sh
go install github.com/ju4n97/hclapi/cmd/hclapi@latest
```

## Verify

```sh
hclapi --help
```
