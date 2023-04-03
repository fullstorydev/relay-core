// This plugin offers a hook that allows tests to observe the requests received
// by the relay. In production, this plugin is not useful.

package test_interceptor_plugin

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/fullstorydev/relay-core/relay/config"
	"github.com/fullstorydev/relay-core/relay/traffic"
)

var (
	Factory    testInterceptorPluginFactory
	pluginName = "test-interceptor"
	logger     = log.New(os.Stdout, fmt.Sprintf("[traffic-%s] ", pluginName), 0)
)

type HandleRequestListener func(request *http.Request)

func NewFactoryWithListener(listener HandleRequestListener) traffic.PluginFactory {
	return testInterceptorPluginFactory{
		listener: listener,
	}
}

type testInterceptorPluginFactory struct {
	listener HandleRequestListener
}

func (f testInterceptorPluginFactory) Name() string {
	return pluginName
}

func (f testInterceptorPluginFactory) New(configFile *config.Section) (traffic.Plugin, error) {
	return &testInterceptorPlugin{
		listener: f.listener,
	}, nil
}

type testInterceptorPlugin struct {
	listener HandleRequestListener
}

func (plug testInterceptorPlugin) Name() string {
	return pluginName
}

func (plug testInterceptorPlugin) HandleRequest(
	response http.ResponseWriter,
	request *http.Request,
	info traffic.RequestInfo,
) bool {
	plug.listener(request)
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
