package main

import (
	"fmt"
	"os"

	"github.com/Axway/agent-sdk/samples/watchclient/pkg/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
