//go:build man

package main

import (
	"os"

	docs "github.com/urfave/cli-docs/v3"
)

func main() {
	cli := newRootCommand()

	man, err := docs.ToMan(cli)
	if err != nil {
		panic(err)
	}

	file, err := os.Create("./man/hclapi.1")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = file.WriteString(man)
	if err != nil {
		panic(err)
	}
}
