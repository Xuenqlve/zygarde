package main

import (
	"context"
	"fmt"
	"os"

	"github.com/xuenqlve/zygarde/internal/cli"
)

func main() {
	if err := cli.Run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
