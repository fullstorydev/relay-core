package relay

import (
	"net/http"

	"github.com/fullstorydev/relay-core/relay/plugins"
)

/*
TrafficService services HTTP requests by calling through to traffic plugins
*/
type TrafficService struct {
	plugins *plugins.Plugins
}

func NewTrafficService(plugs *plugins.Plugins) *TrafficService {
	return &TrafficService{
		plugins: plugs,
	}
}

func (service *TrafficService) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	serviced := false
	for _, trafficPlugin := range service.plugins.Traffic {
		if trafficPlugin.HandleRequest(response, request, serviced) {
			serviced = true
		}
	}
	if serviced == false {
		http.NotFound(response, request)
	}
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
