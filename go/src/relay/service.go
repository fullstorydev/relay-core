package relay

import (
	"fmt"
	"io"
	"net"
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

	// TODO add a control/monitoring service

	return &Service{
		trafficService: trafficService,
		mux:            mux,
		plugins:        plugs,
	}
}

func (service *Service) Start(port int) (io.Closer, int, error) {
	address := fmt.Sprintf("0.0.0.0:%v", port)
	server := &http.Server{
		Addr:    address,
		Handler: service.mux,
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, -1, err
	}

	go func() {
		server.Serve(
			TcpKeepAliveListener{
				listener.(*net.TCPListener),
			},
		)
	}()

	return listener, listener.Addr().(*net.TCPAddr).Port, nil
}
