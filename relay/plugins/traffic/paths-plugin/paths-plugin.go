package paths_plugin

/*
	The Paths plugin watches incoming traffic and optionally rewrites request URL paths.
	The most common use is to remove or rewrite a path prefix.
*/

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/fullstorydev/relay-core/relay/commands"
	"github.com/fullstorydev/relay-core/relay/traffic"
)

var (
	Factory    pathsPluginFactory
	logger     = log.New(os.Stdout, "[traffic-paths] ", 0)
	pluginName = "Paths"
)

type pathsPluginFactory struct{}

func (f pathsPluginFactory) Name() string {
	return pluginName
}

func (f pathsPluginFactory) New(env *commands.Environment) (traffic.Plugin, error) {
	plugin := &pathsPlugin{}

	// Read the replacement value, which is just a literal string. If it's not
	// present, we don't have anything to do, so we just disable the plugin.
	if replacement, ok := env.LookupOptional("TRAFFIC_PATHS_REPLACEMENT"); !ok {
		return nil, nil
	} else {
		plugin.replacement = replacement
	}

	// Read the match value, which is a Go regular expression. Since we know a
	// replacement value was specified, we treat the match value as required;
	// one doesn't make sense without the other.
	if err := env.ParseRequired("TRAFFIC_PATHS_MATCH", func(key string, value string) error {
		if match, err := regexp.Compile(value); err != nil {
			return fmt.Errorf("Could not compile regular expression: %v", err)
		} else {
			plugin.match = match
			return nil
		}
	}); err != nil {
		return nil, err
	}

	logger.Printf("Paths plugin will replace all \"%s\" with \"%s\"", plugin.match, plugin.replacement)

	return plugin, nil
}

type pathsPlugin struct {
	match       *regexp.Regexp
	replacement string
}

func (plug pathsPlugin) Name() string {
	return pluginName
}

func (plug pathsPlugin) HandleRequest(response http.ResponseWriter, request *http.Request, serviced bool) bool {
	if plug.match == nil {
		return false
	}
	request.URL.Path = plug.match.ReplaceAllString(request.URL.Path, plug.replacement)
	return false
}

/*
Copyright 2020 FullStory, Inc.

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
