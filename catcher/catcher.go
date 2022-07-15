package catcher

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"golang.org/x/net/websocket"
)

var logger = log.New(os.Stdout, "[catcher] ", 0)
var ServicePort int = 12346

// Service is an instance of the catcher service. This service is used to test
// the relay. It exposes an HTTP server that captures the last request it
// receives and makes it available via the LastRequest() and LastRequestBody()
// methods. For websocket testing, the /echo endpoint exposes a simple websocket
// server that echoes back whatever it receives.
type Service struct {
	lastRequest []byte
	listener    net.Listener
	mux         *http.ServeMux
}

func NewService() *Service {
	service := &Service{}

	service.mux = http.NewServeMux()
	service.mux.Handle("/echo", websocket.Handler(EchoServer))
	service.mux.HandleFunc("/favicon.ico", func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusNotFound)
		response.Write([]byte("No favicon"))
	})
	service.mux.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
		response.Write([]byte(IndexHTML))

		lastRequest, _ := httputil.DumpRequest(request, true)
		service.lastRequest = lastRequest

		logger.Println("Caught:", request.URL)
	})

	return service
}

func (service *Service) Close() error {
	if service.listener == nil {
		return nil
	}
	return service.listener.Close()
}

func (service *Service) HttpUrl() string {
	if service.listener == nil {
		return ""
	}
	addr := service.listener.Addr().(*net.TCPAddr).String()
	return fmt.Sprintf("http://%v", addr)
}

func (service *Service) LastRequest() (*http.Request, error) {
	if service.lastRequest == nil {
		return nil, errors.New("No last request available")
	}
	return http.ReadRequest(bufio.NewReader(bytes.NewReader(service.lastRequest)))
}

func (service *Service) LastRequestBody() ([]byte, error) {
	request, err := service.LastRequest()
	if err != nil {
		return nil, err
	}

	defer request.Body.Close()
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (service *Service) Port() int {
	if service.listener == nil {
		return 0
	}
	return service.listener.Addr().(*net.TCPAddr).Port
}

func (service *Service) Start(host string, port int) error {
	address := fmt.Sprintf("%v:%v", host, port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	service.listener = listener

	server := &http.Server{
		Addr:    address,
		Handler: service.mux,
	}

	go func() {
		server.Serve(
			tcpKeepAliveListener{
				listener.(*net.TCPListener),
			},
		)
	}()

	return nil
}

// Echo the data received on the WebSocket.
func EchoServer(ws *websocket.Conn) {
	io.Copy(ws, ws)
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(30 * time.Second)
	return tc, nil
}
