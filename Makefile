export PROJECT_HOME := $(shell pwd)
export DIST_PATH     := $(PROJECT_HOME)/dist
export GOPATH       := $(PROJECT_HOME)/go
export GOSRC       	:= $(GOPATH)/src
export GOPKG       	:= $(GOPATH)/pkg

.PHONY: all prep plugins cli compile test clean

all: prep plugins cli

prep:
	go get golang.org/x/net/websocket

plugins:
	mkdir -p $(DIST_PATH)/plugins/traffic/active
	mkdir -p $(DIST_PATH)/plugins/traffic/inactive
	go build -buildmode=plugin -o $(DIST_PATH)/plugins/traffic/active/010-relay.so $(GOSRC)/relay/plugins/traffic/relay/main/main.go
	go build -buildmode=plugin -o $(DIST_PATH)/plugins/traffic/active/020-monitor.so $(GOSRC)/relay/plugins/traffic/monitor/main/main.go
	go build -buildmode=plugin -o $(DIST_PATH)/plugins/traffic/active/030-logging.so $(GOSRC)/relay/plugins/traffic/logging/main/main.go

cli:
	go build -o $(DIST_PATH)/relay $(GOSRC)/relay/main/main.go
	go build -o $(DIST_PATH)/catcher $(GOSRC)/catcher/main/main.go

compile: plugins cli

test: prep compile
	go test relay/...

clean:
	rm -rf $(DIST_PATH)/*
	rm -rf $(GOPKG)/*
