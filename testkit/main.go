package main

import (
	"os"

	"github.com/docker/docker-e2e/testkit/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
