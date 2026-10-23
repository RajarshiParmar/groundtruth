package main

import (
	"fmt"
	"os"

	"github.com/RajarshiParmar/groundtruth/cmd/groundtruth/commands"
)

func main() {
	if err := commands.RootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
