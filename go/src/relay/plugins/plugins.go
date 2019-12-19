package plugins

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"plugin"

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
		return errors.New("Loaded plugin symbol does not implement the TrafficPlugin interface")
	}

	plugins.Traffic = append(plugins.Traffic, trafficPlugin)
	return nil
}
