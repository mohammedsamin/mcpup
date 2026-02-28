package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/mohammedsamin/mcpup/internal/cli"
	"github.com/mohammedsamin/mcpup/internal/core"
)

func main() {
	if err := cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		var exitErr core.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}
