package relay

import (
	"fmt"
	"net"
	"net/http"

	"github.com/fullstorydev/relay-core/relay/plugins"
)

var MonitorPath = "/__relay__up__/"

/*
Service holds the ServMux for the monitoring page as well as traffic plugins
*/
type Service struct {
	listener       net.Listener
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

func (service *Service) Address() string {
	if service.listener == nil {
		return ""
	}
	return service.listener.Addr().(*net.TCPAddr).String()
}

func (service *Service) Close() error {
	if service.listener == nil {
		return nil
	}
	return service.listener.Close()
}

func (service *Service) HttpUrl() string {
	return fmt.Sprintf("http://%v", service.Address())
}

func (service *Service) Port() int {
	if service.listener == nil {
		return 0
	}
	return service.listener.Addr().(*net.TCPAddr).Port
}

func (service *Service) Start(host string, port int) error {
	address := fmt.Sprintf("%v:%v", host, port)
	server := &http.Server{
		Addr:    address,
		Handler: service.mux,
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	service.listener = listener

	go func() {
		server.Serve(
			TcpKeepAliveListener{
				listener.(*net.TCPListener),
			},
		)
	}()

	return nil
}

func (service *Service) WsUrl() string {
	return fmt.Sprintf("ws://%v", service.Address())
}
