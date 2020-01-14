package plugins

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"plugin"

	"relay/commands"
	"relay/plugins/traffic"
)

var logger = log.New(os.Stdout, "[plugin] ", 0)

const PluginDirActiveTrafficPath = "/traffic/active"

type Plugins struct {
	Traffic []traffic.TrafficPlugin
}

func New() *Plugins {
	return &Plugins{}
}

func (plugins *Plugins) Load(dirPath string) error {

	// Load traffic plugins
	activeTrafficPath := path.Join(dirPath, PluginDirActiveTrafficPath)
	soNames, err := readSharedObjectNames(activeTrafficPath)
	if err != nil {
		return err
	}
	for _, soName := range soNames {
		err = plugins.LoadTrafficPlugin(soName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (plugins *Plugins) TrafficEnvVars() []commands.EnvVar {
	vars := []commands.EnvVar{}
	for _, trafficPlugin := range plugins.Traffic {
		for name, required := range trafficPlugin.ConfigVars() {
			vars = append(vars, commands.EnvVar{name, required, ""})
		}
	}
	return vars
}

func (plugins *Plugins) SetupEnvironment() error {
	err := commands.SetupEnvironment(plugins.TrafficEnvVars(), []string{})
	if err != nil {
		return err
	}
	for _, trafficPlugin := range plugins.Traffic {
		if trafficPlugin.Config() == false {
			return errors.New(fmt.Sprintf("Traffic plugin \"%v\" configuration error.", trafficPlugin.Name()))
		}
	}
	return nil
}

func (plugins *Plugins) PrintEnvUsage() {
	commands.PrintEnvUsage(plugins.TrafficEnvVars())
}

func (plugins *Plugins) LoadTrafficPlugin(pluginPath string) error {
	plug, err := plugin.Open(pluginPath)
	if err != nil {
		return err
	}
	symTrafficPlugin, err := plug.Lookup("Plugin")
	if err != nil {
		return err
	}

	var trafficPlugin traffic.TrafficPlugin
	trafficPlugin, ok := symTrafficPlugin.(traffic.TrafficPlugin)
	if !ok {
		return errors.New(fmt.Sprintf("Plugin does not implement the TrafficPlugin interface:\n\t%v", pluginPath))
	}

	plugins.Traffic = append(plugins.Traffic, trafficPlugin)
	return nil
}

func readSharedObjectNames(dirPath string) ([]string, error) {
	pathInfo, err := os.Stat(dirPath)
	results := []string{}
	if err != nil {
		return results, err
	}
	if pathInfo.IsDir() == false {
		return results, errors.New(fmt.Sprintf("Path is not a directory %v", dirPath))
	}
	return filepath.Glob(path.Join(dirPath, "*.so"))
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
