package relay_plugin_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/fullstorydev/relay-core/catcher"
	"github.com/fullstorydev/relay-core/relay"
	"github.com/fullstorydev/relay-core/relay/commands"
	"github.com/fullstorydev/relay-core/relay/test"
	"golang.org/x/net/websocket"
)

func TestBasicRelay(t *testing.T) {
	test.WithCatcherAndRelay(t, nil, nil, func(catcherService *catcher.Service, relayService *relay.Service) {
		catcherBody := getBody(catcherService.HttpUrl(), t)
		if catcherBody == nil {
			return
		}

		relayBody := getBody(relayService.HttpUrl(), t)
		if relayBody == nil {
			return
		}

		if bytes.Equal(catcherBody, relayBody) == false {
			t.Errorf("Bodies don't match: \"%v\" \"%v\"", catcherBody, relayBody)
			return
		}
	})
}

func TestOriginOverride(t *testing.T) {
	newOrigin := "example.com"
	env := commands.Environment{
		"TRAFFIC_RELAY_ORIGIN_OVERRIDE": newOrigin,
	}

	test.WithCatcherAndRelay(t, env, nil, func(catcherService *catcher.Service, relayService *relay.Service) {
		_, err := http.Get(relayService.HttpUrl())
		if err != nil {
			t.Errorf("Error GETing: %v", err)
			return
		}

		lastRequest, err := catcherService.LastRequest()
		if err != nil {
			t.Errorf("Error reading last request from catcher: %v", err)
			return
		}

		lastRequestOrigin := lastRequest.Header.Get("Origin")
		if "http://"+newOrigin != lastRequestOrigin {
			t.Errorf("Origin override mismatch: \"%v\" \"%v\"", newOrigin, lastRequestOrigin)
			return
		}
	})
}

func TestMaxBodySize(t *testing.T) {
	env := commands.Environment{
		"TRAFFIC_RELAY_MAX_BODY_SIZE": fmt.Sprintf("%v", 5),
	}

	test.WithCatcherAndRelay(t, env, nil, func(catcherService *catcher.Service, relayService *relay.Service) {
		response, err := http.Get(relayService.HttpUrl())
		if err != nil {
			t.Errorf("Error GETing: %v", err)
			return
		}
		defer response.Body.Close()
		if response.StatusCode != 503 {
			t.Errorf("Expected 503 response for surpassing max body size: %v", response)
			return
		}
	})
}

func TestRelayNotFound(t *testing.T) {
	test.WithCatcherAndRelay(t, nil, nil, func(catcherService *catcher.Service, relayService *relay.Service) {
		faviconURL := fmt.Sprintf("%v/favicon.ico", relayService.HttpUrl())
		response, err := http.Get(faviconURL)
		if err != nil {
			t.Errorf("Error GETing: %v", err)
			return
		}
		if response.StatusCode != 404 {
			t.Errorf("Should have received 404: %v", response)
			return
		}
	})
}

func TestWebSocketEcho(t *testing.T) {
	test.WithCatcherAndRelay(t, nil, nil, func(catcherService *catcher.Service, relayService *relay.Service) {
		echoURL := fmt.Sprintf("%v/echo", relayService.WsUrl())
		ws, err := websocket.Dial(echoURL, "", relayService.HttpUrl())
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
	})
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
