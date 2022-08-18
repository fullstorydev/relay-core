package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/fullstorydev/relay-core/relay"
	"github.com/fullstorydev/relay-core/relay/commands"
	"github.com/fullstorydev/relay-core/relay/plugins"
)

var logger = log.New(os.Stdout, "[relay] ", 0)

// These are the env variables in play
var RelayPortVar = "RELAY_PORT"

func main() {
	envProvider := commands.NewDefaultEnvironmentProvider()
	env, err := commands.GetEnvironmentOrPrintUsage(envProvider, []commands.EnvVar{
		{
			EnvKey:   RelayPortVar,
			Required: true,
		},
	})
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	parsedPort, err := strconv.ParseInt(env[RelayPortVar], 10, 32)
	if err != nil {
		logger.Printf("Error parsing relay port: \"%v\"", env[RelayPortVar])
		os.Exit(1)
	}
	relayPort := int(parsedPort)

	trafficPlugins, err := plugins.Load(plugins.Default, envProvider)
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	logger.Println("Plugins:")
	for _, tp := range trafficPlugins {
		logger.Println("\tTraffic:", tp.Name())
	}

	relayService := relay.NewService(trafficPlugins)
	if err := relayService.Start("0.0.0.0", relayPort); err != nil {
		panic("Could not start catcher service: " + err.Error())
	}
	logger.Println("Relay listening on port", relayService.Port())
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
