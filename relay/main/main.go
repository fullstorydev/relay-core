package main

import (
	"flag"
	"io"
	"log"
	"os"
	"time"

	"github.com/fullstorydev/relay-core/relay"
	"github.com/fullstorydev/relay-core/relay/config"
	"github.com/fullstorydev/relay-core/relay/environment"
	plugin_loader "github.com/fullstorydev/relay-core/relay/traffic/plugin-loader"
)

var logger = log.New(os.Stdout, "[relay] ", 0)

func readConfigFile(path string) (rawConfigFileBytes []byte, err error) {
	if path == "-" {
		rawConfigFileBytes, err = io.ReadAll(os.Stdin)
		return
	}

	rawConfigFileBytes, err = os.ReadFile(path)
	return
}

func main() {
	// The --config option determines the path to the configuration file. A
	// default configuration file, 'relay.yaml', is distributed with the relay,
	// so it's not necessary to specify one if you just want to configure the
	// relay with environment variables. Use '-' to read the configuration file
	// from stdin.
	configFilePath := flag.String("config", "relay.yaml", "Configuration file path")
	flag.Parse()

	rawConfigFileBytes, err := readConfigFile(*configFilePath)
	if err != nil {
		logger.Printf(`Couldn't read configuration file "%s": %v\n`, *configFilePath, err)
		os.Exit(1)
	}

	// Substitute the values of environment variables into the configuration
	// file. In versions of the relay prior to 0.3, configuration was performed
	// entirely via environment variables. Environment variable substitution
	// allows configurations based on those older environment variables to
	// continue to work and generally increases the flexibility of the
	// configuration file.
	envProvider := environment.NewDefaultProvider()
	env := environment.NewMap(envProvider)
	configFileString := env.SubstituteVarsIntoYaml(string(rawConfigFileBytes))

	// Parse the configuration file.
	configFile, err := config.NewFileFromYamlString(configFileString)
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	config, err := relay.ReadOptions(configFile)
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	trafficPlugins, err := plugin_loader.Load(plugin_loader.DefaultPlugins, configFile)
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	logger.Println("Active plugins:")
	for _, tp := range trafficPlugins {
		logger.Println("\tTraffic:", tp.Name())
	}

	relayService := relay.NewService(config.Relay, trafficPlugins)
	if err := relayService.Start("0.0.0.0", config.Service.Port); err != nil {
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
