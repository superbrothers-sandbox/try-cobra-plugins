package main

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{}
	cmd.Use = "cli <command>"

	loadPlugins(cmd, os.Stdin, os.Stdout)

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
