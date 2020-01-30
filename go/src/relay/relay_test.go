package relay

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"golang.org/x/net/websocket"

	"catcher"
	"relay/plugins"
)

func TestBasicRelay(t *testing.T) {
	catcherCloser, catcherPort, relayCloser, relayPort, err := setupCatcherAndRelay()
	if err != nil {
		t.Errorf("Error starting catcher and relay: %v", err)
	}
	defer catcherCloser.Close()
	defer relayCloser.Close()

	catcherURL := fmt.Sprintf("http://127.0.0.1:%v", catcherPort)
	catcherBody := getBody(catcherURL, t)
	if catcherBody == nil {
		return
	}

	relayURL := fmt.Sprintf("http://127.0.0.1:%v", relayPort)
	relayBody := getBody(relayURL, t)
	if relayBody == nil {
		return
	}

	if bytes.Equal(catcherBody, relayBody) == false {
		t.Errorf("Bodies don't match: \"%v\" \"%v\"", catcherBody, relayBody)
		return
	}
}

func TestMaxBodySize(t *testing.T) {
	os.Setenv("TRAFFIC_RELAY_MAX_BODY_SIZE", fmt.Sprintf("%v", 5))
	defer os.Setenv("TRAFFIC_RELAY_MAX_BODY_SIZE", "2097152") // Unsetenv doesn't work
	catcherCloser, _, relayCloser, relayPort, err := setupCatcherAndRelay()
	if err != nil {
		t.Errorf("Error starting catcher and relay: %v", err)
		return
	}
	defer catcherCloser.Close()
	defer relayCloser.Close()

	relayURL := fmt.Sprintf("http://127.0.0.1:%v", relayPort)

	response, err := http.Get(relayURL)
	if err != nil {
		t.Errorf("Error GETing: %v", err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode != 503 {
		t.Errorf("Expected 503 response for surpassing max body size: %v", response)
		return
	}
}

func TestRelayNotFound(t *testing.T) {
	catcherCloser, _, relayCloser, relayPort, err := setupCatcherAndRelay()
	if err != nil {
		t.Errorf("Error starting catcher and relay: %v", err)
		return
	}
	defer catcherCloser.Close()
	defer relayCloser.Close()

	faviconURL := fmt.Sprintf("http://127.0.0.1:%v/favicon.ico", relayPort)
	response, err := http.Get(faviconURL)
	if err != nil {
		t.Errorf("Error GETing: %v", err)
		return
	}
	if response.StatusCode != 404 {
		t.Errorf("Should have received 404: %v", response)
		return
	}
}

func TestWebSocketEcho(t *testing.T) {
	catcherCloser, _, relayCloser, relayPort, err := setupCatcherAndRelay()
	if err != nil {
		t.Errorf("Error starting catcher and relay: %v", err)
		return
	}
	defer catcherCloser.Close()
	defer relayCloser.Close()

	echoURL := fmt.Sprintf("ws://127.0.0.1:%v/echo", relayPort)
	originURL := fmt.Sprintf("http://127.0.0.1:%v/", relayPort)

	ws, err := websocket.Dial(echoURL, "", originURL)
	if err != nil {
		t.Errorf("Error dialing websocket: %v", err)
		return
	}
	err = testEcho(ws, "Come in, good buddy")
	if err != nil {
		t.Errorf("Error in echo: %v", err)
		return
	}
	err = testEcho(ws, "10-4, Rocket")
	if err != nil {
		t.Errorf("Error in second echo: %v", err)
		return
	}
}

func setupCatcherAndRelay() (catcherCloser io.Closer, catcherPort int, relayCloser io.Closer, relayPort int, err error) {
	catcherCloser, catcherPort, err = catcher.Start(0)
	if err != nil {
		return nil, -1, nil, -1, err
	}
	catcherURL := fmt.Sprintf("http://127.0.0.1:%v", catcherPort)
	os.Setenv("TRAFFIC_RELAY_TARGET", catcherURL)
	relayService, err := setupRelay()
	if err != nil {
		catcherCloser.Close()
		return nil, -1, nil, -1, err
	}
	relayCloser, relayPort, err = relayService.Start(0)
	if err != nil {
		catcherCloser.Close()
		return nil, -1, nil, -1, err
	}
	return catcherCloser, catcherPort, relayCloser, relayPort, nil
}

func setupRelay() (*Service, error) {
	var pluginsPath string = "../../../dist/plugins"

	plugs := plugins.New()
	err := plugs.Load(pluginsPath)
	if err != nil {
		return nil, err
	}
	err = plugs.SetupEnvironment()
	if err != nil {
		return nil, err
	}
	return NewService(plugs), nil
}

func testEcho(conn *websocket.Conn, message string) error {
	_, err := conn.Write([]byte(message))
	if err != nil {
		return err
	}
	var response = make([]byte, len(message)+10)
	n, err := conn.Read(response)
	if err != nil {
		return err
	}
	if strings.Compare(message, string(response[:n])) != 0 {
		return errors.New(fmt.Sprintf("Unexpected echo response: %v", string(response[:n])))
	}
	return nil
}

func getBody(url string, t *testing.T) []byte {
	response, err := http.Get(url)
	if err != nil {
		t.Errorf("Error GETing: %v", err)
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Errorf("Non-200 GET: %v", response)
		return nil
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Errorf("Error GETing body: %v", err)
		return nil
	}
	return body
}
