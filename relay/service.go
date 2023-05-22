package relay

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/fullstorydev/relay-core/relay/traffic"
)

var MonitorPath = "/__relay__up__/"

// ServiceOptions contains configuration options for the relay network service.
//
// See also traffic.RelayOptions, which provides options for the actual relay
// functionality.
type ServiceOptions struct {
	Port int // The port that the relay service should listen on.
}

func NewDefaultServiceOptions() *ServiceOptions {
	return &ServiceOptions{}
}

// Service implements the relay service, exposing both the traffic handler and
// the monitoring page.
type Service struct {
	listener net.Listener
	mux      *http.ServeMux
}

func NewService(relayConfig *traffic.RelayOptions, trafficPlugins []traffic.Plugin) *Service {
	mux := http.NewServeMux()

	// Write a simple page for monitoring.
	// TODO add a control/monitoring service
	mux.HandleFunc(MonitorPath, func(response http.ResponseWriter, request *http.Request) {
		response.Header().Add("Content-Type", "text/html")
		response.Write([]byte("<html><body>Up</body></html>"))
	})

	// Set up the traffic handler.
	mux.Handle("/", traffic.NewHandler(relayConfig, trafficPlugins))

	return &Service{
		mux: mux,
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
		Addr:              address,
		Handler:           service.mux,
		ReadHeaderTimeout: 2 * time.Second,
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
