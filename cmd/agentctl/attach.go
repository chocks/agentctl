package main

import (
	"fmt"
	"os"

	"github.com/chocks/agentctl/pkg/config"
)

func cmdAttach(paths config.Paths) {
	fmt.Fprintln(os.Stderr, "agentctl attach: not yet implemented")
	os.Exit(1)
}

func cmdDetach() {
	fmt.Fprintln(os.Stderr, "agentctl detach: not yet implemented")
	os.Exit(1)
}
