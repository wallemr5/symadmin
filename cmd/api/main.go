package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"gitlab.dmall.com/arch/sym-admin/cmd/api/app"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	rootCmd := app.GetRootCmd(os.Args[1:])

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(-1)
	}
}
