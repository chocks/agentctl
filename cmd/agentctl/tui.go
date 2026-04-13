package main

import (
	"fmt"
	"os"

	"github.com/chocks/agentctl/pkg/config"
)

func cmdTUI(paths config.Paths) {
	fmt.Fprintln(os.Stderr, "agentctl ui: not yet implemented")
	os.Exit(1)
}
