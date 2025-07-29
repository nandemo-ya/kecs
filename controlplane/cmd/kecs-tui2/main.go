package main

import (
	"fmt"
	"os"

	"github.com/nandemo-ya/kecs/controlplane/internal/tui2"
)

func main() {
	if err := tui2.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}