export PROJECT_HOME := $(shell pwd)
export DIST_PATH     := $(PROJECT_HOME)/dist
export GOPATH       := $(PROJECT_HOME)/go
export GOSRC       	:= $(GOPATH)/src
export GOPKG       	:= $(GOPATH)/pkg

.PHONY: compile test clean

compile:
	mkdir -p $(DIST_PATH)/plugins/traffic/active
	mkdir -p $(DIST_PATH)/plugins/traffic/inactive
	# Write XXX-plugin.so where XXX is a number so the plugins are loaded in a set order 
	go build -buildmode=plugin -o $(DIST_PATH)/plugins/traffic/active/010-logging.so $(GOSRC)/relay/plugins/traffic/logging/main/main.go
	go build -buildmode=plugin -o $(DIST_PATH)/plugins/traffic/active/020-relay.so $(GOSRC)/relay/plugins/traffic/relay/main/main.go
	go build -buildmode=plugin -o $(DIST_PATH)/plugins/traffic/active/030-monitor.so $(GOSRC)/relay/plugins/traffic/monitor/main/main.go
	go build -o $(DIST_PATH)/relay $(GOSRC)/relay/main/main.go

test:
	go test relay/...

clean:
	rm -rf $(DIST_PATH)/*
	rm -rf $(GOPKG)/*
