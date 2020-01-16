package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/websocket"
)

var logger = log.New(os.Stdout, "[catcher] ", 0)
var WebSocketPort int64 = 12346

func main() {
	http.Handle("/echo", websocket.Handler(EchoServer))
	http.HandleFunc("/favicon.ico", func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusNotFound)
		response.Write([]byte("No favicon"))
	})
	http.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
		response.Write([]byte("Catcher is live"))
	})
	logger.Println("Catcher listening on port", WebSocketPort)
	err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", WebSocketPort), nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

// Echo the data received on the WebSocket.
func EchoServer(ws *websocket.Conn) {
	io.Copy(ws, ws)
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
