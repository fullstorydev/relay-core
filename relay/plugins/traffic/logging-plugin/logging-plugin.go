package logging_plugin

import (
	"log"
	"net/http"
	"os"

	"github.com/fullstorydev/relay-core/relay/plugins/traffic"
)

var (
	Factory    loggingPluginFactory
	logger     = log.New(os.Stdout, "[traffic-logging] ", 0)
	pluginName = "Logging"
)

type loggingPluginFactory struct{}

func (f loggingPluginFactory) Name() string {
	return pluginName
}

func (f loggingPluginFactory) ConfigVars() map[string]bool {
	return map[string]bool{}
}

func (f loggingPluginFactory) New(env map[string]string) (traffic.Plugin, error) {
	return &loggingPlugin{}, nil
}

type loggingPlugin struct{}

func (plug *loggingPlugin) Name() string {
	return pluginName
}

func (plug *loggingPlugin) HandleRequest(response http.ResponseWriter, request *http.Request, serviced bool) bool {
	servicedMessage := "serviced"
	if serviced == false {
		servicedMessage = "not serviced"
	}

	logger.Printf("%s %s %s: %s", request.Method, request.Host, request.URL, servicedMessage)
	return false
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
