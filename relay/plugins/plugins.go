package plugins

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"plugin"

	"github.com/fullstorydev/relay-core/relay/commands"
	"github.com/fullstorydev/relay-core/relay/plugins/traffic"
)

var logger = log.New(os.Stdout, "[plugin] ", 0)

const PluginDirActiveTrafficPath = "/traffic/active"

// Plugins loads, configures, and manages dynamically loaded plugins. Most relay
// functionality is implemented via these plugins.
//
// The usual flow is to load the plugins via Load() or LoadWithFilter(),
// configure them via SetupEnvironment() or ConfigurePlugins(), and then
// construct the relay Service using the fully-configured Plugins instance. See
// the documentation of each method for more details about when to use them.
type Plugins struct {
	Factories []traffic.PluginFactory
	Traffic   []traffic.Plugin
}

func New() *Plugins {
	return &Plugins{}
}

// Load loads all available plugins in the provided directory.
func (plugins *Plugins) Load(dirPath string) error {
	return plugins.LoadWithFilter(dirPath, nil)
}

// LoadWithFilter loads plugins from the provided directory if they're present
// in the provided allowlist. The allowlist is a set where the keys are values
// returned from traffic.PluginFactory#Name().
func (plugins *Plugins) LoadWithFilter(dirPath string, pluginAllowlist map[string]bool) error {
	// Load traffic plugins
	activeTrafficPath := path.Join(dirPath, PluginDirActiveTrafficPath)
	soNames, err := readSharedObjectNames(activeTrafficPath)
	if err != nil {
		return err
	}
	for _, soName := range soNames {
		err = plugins.loadTrafficPlugin(soName, pluginAllowlist)
		if err != nil {
			return err
		}
	}

	return nil
}

// TrafficEnvVars returns a list of the configuration variables each loaded
// plugin uses.
func (plugins *Plugins) TrafficEnvVars() []commands.EnvVar {
	vars := []commands.EnvVar{}
	for _, trafficPluginFactory := range plugins.Factories {
		for name, required := range trafficPluginFactory.ConfigVars() {
			vars = append(vars, commands.EnvVar{
				EnvKey:     name,
				Required:   required,
				DefaultVal: "",
				IsDir:      false,
			})
		}
	}
	return vars
}

// SetupEnvironment configures the loaded plugins. It reads configuration
// variables from the environment and from any .env file that may be present.
func (plugins *Plugins) SetupEnvironment() (commands.Environment, error) {
	vars := plugins.TrafficEnvVars()

	env, err := commands.GetEnvironment(vars)
	if err != nil {
		return env, err
	}

	if err := plugins.ConfigurePlugins(env); err != nil {
		return env, err
	}

	return env, nil
}

// ConfigurePlugins configures the loaded plugins using the provided
// configuration variables. It doesn't consult the environment or any other
// external source at all. This is the preferred method to use in tests.
func (plugins *Plugins) ConfigurePlugins(env commands.Environment) error {
	for _, trafficPluginFactory := range plugins.Factories {
		plugin, err := trafficPluginFactory.New(env)
		if err != nil {
			return fmt.Errorf("Traffic plugin \"%v\" configuration error: %v", trafficPluginFactory.Name(), err)
		}

		if plugin == nil {
			continue // This plugin is inactive.
		}

		plugins.Traffic = append(plugins.Traffic, plugin)
	}

	return nil
}

func (plugins *Plugins) loadTrafficPlugin(pluginPath string, pluginAllowlist map[string]bool) error {
	plug, err := plugin.Open(pluginPath)
	if err != nil {
		return err
	}
	symTrafficPluginFactory, err := plug.Lookup("Factory")
	if err != nil {
		return err
	}

	var trafficPluginFactory traffic.PluginFactory
	trafficPluginFactory, ok := symTrafficPluginFactory.(traffic.PluginFactory)
	if !ok {
		return fmt.Errorf("Factory does not implement the TrafficPluginFactory interface:\n\t%v", pluginPath)
	}

	if pluginAllowlist != nil && !pluginAllowlist[trafficPluginFactory.Name()] {
		// Skip this plugin. (Note that this isn't an error; we're just
		// declining to load a plugin we could've loaded.)
		logger.Printf("Skipping plugin: %s\n", trafficPluginFactory.Name())
		return nil
	}

	logger.Printf("Loading plugin: %s\n", trafficPluginFactory.Name())
	plugins.Factories = append(plugins.Factories, trafficPluginFactory)

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
