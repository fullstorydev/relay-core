package catcher

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/websocket"
)

var logger = log.New(os.Stdout, "[catcher] ", 0)
var ServicePort int = 12346

func Start(port int) (io.Closer, int, error) {
	mux := http.NewServeMux()
	mux.Handle("/echo", websocket.Handler(EchoServer))
	mux.HandleFunc("/favicon.ico", func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusNotFound)
		response.Write([]byte("No favicon"))
	})
	mux.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
		response.Write([]byte("Catcher is online"))
	})

	address := fmt.Sprintf("0.0.0.0:%v", port)

	server := &http.Server{
		Addr:    address,
		Handler: mux,
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, -1, err
	}

	go func() {
		server.Serve(
			tcpKeepAliveListener{
				listener.(*net.TCPListener),
			},
		)
	}()

	return listener, listener.Addr().(*net.TCPAddr).Port, nil
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
