package main

import (
	"fmt"

	"github.com/sgotti/acido/version"
)

var cmdVersion = &Command{
	Name:        "version",
	Description: "Print the version and exit",
	Summary:     "Print the version and exit",
	Run:         runVersion,
}

func init() {
	commands = append(commands, cmdVersion)
}

func runVersion(args []string) (exit int) {
	fmt.Printf("acido version %s\n", version.Version)
	return
}
