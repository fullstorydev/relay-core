package traffic

import (
	"fmt"

	"github.com/fullstorydev/relay-core/relay/commands"
)

// Load creates and configures a set of traffic plugins.
func LoadPlugins(
	pluginFactories []PluginFactory,
	env *commands.Environment,
) ([]Plugin, error) {
	trafficPlugins := []Plugin{}

	for _, factory := range pluginFactories {
		logger.Printf("Loading plugin: %s\n", factory.Name())

		plugin, err := factory.New(env)
		if err != nil {
			return nil, fmt.Errorf("Traffic plugin \"%v\" configuration error: %v", factory.Name(), err)
		}

		if plugin == nil {
			continue // This plugin is inactive.
		}

		trafficPlugins = append(trafficPlugins, plugin)
	}

	return trafficPlugins, nil
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
