package main

import (
	"fmt"
	"os"

	"github.com/kamrankamilli/gsvnc/pkg/cli"
)

func main() {
	if err := cli.RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
