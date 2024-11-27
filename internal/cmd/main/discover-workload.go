package main

import (
	"context"
	"os"

	"github.com/opdev/discover-workload/internal/cmd/discoverworkload"
)

func main() {
	if err := discoverworkload.NewCommand(context.Background()).Execute(); err != nil {
		os.Exit(1)
	}
}
