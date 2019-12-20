package relay

import (
	"fmt"
	"net/http"

	"relay/plugins"
)

var MonitorPath = "/__relay__up__/"

/*
Service holds the ServMux for the monitoring page as well as traffic plugins
*/
type Service struct {
	trafficService *TrafficService
	mux            *http.ServeMux
	plugins        *plugins.Plugins
}

func NewService(plugs *plugins.Plugins) *Service {
	mux := http.NewServeMux()

	// Write a simple page for monitoring
	mux.HandleFunc(MonitorPath, func(response http.ResponseWriter, request *http.Request) {
		response.Header().Add("Content-Type", "text/html")
		response.Write([]byte("<html><body>Up</body></html>"))
	})

	// Handle everything else with traffic plugins
	trafficService := NewTrafficService(plugs)
	mux.Handle("/", trafficService)

	return &Service{
		trafficService: trafficService,
		mux:            mux,
		plugins:        plugs,
	}
}

/*
Starts and blocks on HTTP service
*/
func (service *Service) Serve(port int64) {
	http.ListenAndServe(fmt.Sprintf(":%d", port), service.mux)
}
