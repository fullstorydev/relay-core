package traffic

import (
	"log"
	"net/http"
	"os"
)

var logger = log.New(os.Stdout, "[relay-traffic] ", 0)

/*
	relay.plugins.Plugins loads traffic plugins in order to service HTTP requests
*/
type TrafficPlugin interface {
	/*
		A human readable name for this plugin, like "Logging" or "Attack detector"
	*/
	Name() string

	/*
		HandleRequest is with an incoming traffic HTTP request
		`serviced` is true if a previously called TrafficPlugin has responded to the request
		returns true iff the response has been fully serviced
	*/
	HandleRequest(response http.ResponseWriter, request *http.Request, serviced bool) bool

	/*
		ConfigVars returns a map of environment variables and whether they are required by this plugin
		For example, if the plugin expects environment variables  and MIN_LENGTH and can't work if MIN_LENGTH is set then the return value would be:

			return map[string]bool{
				"MAX_COUNT":   false,
				"MIN_LENGTH":  true,
			}

	*/
	ConfigVars() map[string]bool

	/*
		Config checks whether the plugin has the environment variables needed to run
	*/
	Config() bool
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
