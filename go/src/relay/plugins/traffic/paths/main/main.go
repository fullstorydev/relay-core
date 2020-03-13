package main

/*
	The Paths plugin watches incoming traffic and optionally rewrites request URL paths.
	The most common use is to remove or rewrite a path prefix.
*/

import (
	"log"
	"net/http"
	"os"
	"regexp"
)

var (
	// This is what relay.plugin.Plugins will load to handle traffic plugin duties
	Plugin         pathsPlugin = New()
	logger                     = log.New(os.Stdout, "[traffic-paths] ", 0)
	matchVar                   = "TRAFFIC_PATHS_MATCH"       // A go regexp string or empty
	replacementVar             = "TRAFFIC_PATHS_REPLACEMENT" // A go repl string or empty
)

type pathsPlugin struct {
	match       *regexp.Regexp
	replacement string
}

func New() pathsPlugin {
	return pathsPlugin{
		match:       nil,
		replacement: "",
	}
}

func (plug pathsPlugin) Name() string {
	return "Paths"
}

func (plug pathsPlugin) HandleRequest(response http.ResponseWriter, request *http.Request, serviced bool) bool {
	if plug.match == nil {
		return false
	}
	request.URL.Path = plug.match.ReplaceAllString(request.URL.Path, plug.replacement)
	return false
}

func (plug pathsPlugin) ConfigVars() map[string]bool {
	return map[string]bool{
		matchVar:       false,
		replacementVar: false,
	}
}

func (plug *pathsPlugin) Config() bool {
	match := os.Getenv(matchVar)
	replacement := os.Getenv(replacementVar)
	if len(match) == 0 && len(replacement) == 0 {
		return true
	}
	if len(match) == 0 {
		logger.Println("Paths plugin has", replacementVar, "but not", matchVar)
		return false
	}

	matchRE, err := regexp.Compile(match)
	if err != nil {
		logger.Println("Could not compile", matchVar, "regular expression:", err)
		return false
	}

	plug.match = matchRE
	plug.replacement = replacement

	logger.Printf("Paths plugin will replace all \"%s\" with \"%s\"", plug.match, plug.replacement)

	return true
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
