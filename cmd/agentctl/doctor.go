package main

import (
	"fmt"
	"os"

	"github.com/chocks/agentctl/pkg/config"
)

func cmdDoctor(paths config.Paths) {
	fmt.Fprintln(os.Stderr, "agentctl doctor: not yet implemented")
	os.Exit(1)
}
