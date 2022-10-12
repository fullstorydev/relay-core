package headers_plugin

// The Headers plugin provides the capability to transform request headers.

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/fullstorydev/relay-core/relay/commands"
	"github.com/fullstorydev/relay-core/relay/traffic"
)

var (
	Factory    headersPluginFactory
	logger     = log.New(os.Stdout, "[traffic-headers] ", 0)
	pluginName = "Headers"
)

type headersPluginFactory struct{}

func (f headersPluginFactory) Name() string {
	return pluginName
}

func (f headersPluginFactory) New(env *commands.Environment) (traffic.Plugin, error) {
	plugin := &headersPlugin{}

	if originOverrideVal, ok := env.LookupOptional("TRAFFIC_RELAY_ORIGIN_OVERRIDE"); !ok {
		return nil, nil
	} else {
		plugin.originOverride = originOverrideVal
	}

	logger.Printf(`Headers plugin will override origin to "%s"`, plugin.originOverride)

	return plugin, nil
}

type headersPlugin struct {
	originOverride string
}

func (plug headersPlugin) Name() string {
	return pluginName
}

func (plug headersPlugin) HandleRequest(
	response http.ResponseWriter,
	request *http.Request,
	info traffic.RequestInfo,
) bool {
	if info.Serviced {
		return false
	}

	request.Header.Set(
		"Origin",
		fmt.Sprintf("%v://%v", request.URL.Scheme, plug.originOverride),
	)

	return false
}

/*
Copyright 2022 FullStory, Inc.

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
