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

	"github.com/fullstorydev/relay-core/relay/plugins/traffic"
)

var (
	Factory    pathsPluginFactory
	logger     = log.New(os.Stdout, "[traffic-paths] ", 0)
	pluginName = "Paths"

	matchVar       = "TRAFFIC_PATHS_MATCH"       // A go regexp string or empty
	replacementVar = "TRAFFIC_PATHS_REPLACEMENT" // A go repl string or empty
)

type pathsPluginFactory struct{}

func (f pathsPluginFactory) Name() string {
	return pluginName
}

func (f pathsPluginFactory) ConfigVars() map[string]bool {
	return map[string]bool{
		matchVar:       false,
		replacementVar: false,
	}
}

func (f pathsPluginFactory) New(env map[string]string) (traffic.Plugin, error) {
	match := env[matchVar]
	replacement := env[replacementVar]
	if len(match) == 0 && len(replacement) == 0 {
		return nil, nil
	}
	if len(match) == 0 {
		return nil, fmt.Errorf("Paths plugin has %v but not %v", replacementVar, matchVar)
	}

	matchRE, err := regexp.Compile(match)
	if err != nil {
		return nil, fmt.Errorf("Could not compile %v regular expression: %v", matchVar, err)
	}

	logger.Printf("Paths plugin will replace all \"%s\" with \"%s\"", matchRE, replacement)

	return &pathsPlugin{
		match:       matchRE,
		replacement: replacement,
	}, nil
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
