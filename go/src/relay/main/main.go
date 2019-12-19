package main

import (
	"log"
	"os"

	"relay/plugins"
)

var logger = log.New(os.Stdout, "[relay] ", 0)

func main() {
	logger.Println("Starting...")

	plugs := plugins.New()
	err := plugs.Load("./plugins")
	if err != nil {
		logger.Println("Error loading plugins", err)
		os.Exit(1)
	}
	logger.Println("Plugins:")
	for _, tp := range plugs.Traffic {
		logger.Println("\tTraffic:", tp.Name())
	}
}
