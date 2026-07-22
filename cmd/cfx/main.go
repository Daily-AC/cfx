package main

import (
	"fmt"
	"os"

	"github.com/e0-7/cfx/internal/app"
)

func main() {
	if err := app.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "cfx:", err)
		os.Exit(1)
	}
}
