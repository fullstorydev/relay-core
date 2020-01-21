package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"relay"
	"relay/commands"
	"relay/plugins"
)

var logger = log.New(os.Stdout, "[relay] ", 0)

// These are the env variables in play
var RelayPortVar = "RELAY_PORT"
var RelayPluginsPathVar = "RELAY_PLUGINS_PATH"

// This defines whether config variables are required
var EnvVars = []commands.EnvVar{
	{RelayPortVar, true, ""},
	{RelayPluginsPathVar, false, "./plugins"},
}

// These config variables, if set, must be paths to valid directories
var DirExistenceVars = []string{
	RelayPluginsPathVar,
}

func main() {
	if commands.SetupEnvironment(EnvVars, DirExistenceVars) != nil {
		commands.PrintEnvUsage(EnvVars)
		os.Exit(1)
	}
	parsedPort, err := strconv.ParseInt(os.Getenv(RelayPortVar), 10, 32)
	if err != nil {
		logger.Printf("Error parsing relay port: \"%v\"", os.Getenv(RelayPortVar))
		os.Exit(1)
	}
	relayPort := int(parsedPort)

	pluginsPath := os.Getenv(RelayPluginsPathVar)

	plugs := plugins.New()
	err = plugs.Load(pluginsPath)
	if err != nil {
		logger.Println(fmt.Sprintf("Error loading plugins:\n\t%v", err))
		os.Exit(1)
	}
	err = plugs.SetupEnvironment()
	if err != nil {
		logger.Println(err)
		plugs.PrintEnvUsage()
		os.Exit(1)
	}
	logger.Println("Plugins:")
	for _, tp := range plugs.Traffic {
		logger.Println("\tTraffic:", tp.Name())
	}

	relayService := relay.NewService(plugs)
	_, port, err := relayService.Start(relayPort)
	if err != nil {
		panic("Could not start catcher service: " + err.Error())
	}
	logger.Println("Relay listening on port", port)
	for {
		time.Sleep(100 * time.Minute)
	}
}

/*
Copyright 2019 FullStory, Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software
and associated documentation files (the "Software"), to deal in the Software without restriction,
including without limitation the rights to use, copy, modify, merge, publish, distribute,
sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or
substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT
NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
