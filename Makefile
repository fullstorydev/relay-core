export PROJECT_HOME := $(shell pwd)
export DIST_PATH     := $(PROJECT_HOME)/dist
export GOPATH       := $(PROJECT_HOME)/go
export GOSRC       	:= $(GOPATH)/src
export GOPKG       	:= $(GOPATH)/pkg

.PHONY: compile test clean

compile:
	mkdir -p $(DIST_PATH)/plugins/traffic/active
	mkdir -p $(DIST_PATH)/plugins/traffic/inactive
	go build -buildmode=plugin -o $(DIST_PATH)/plugins/traffic/active/logging.so $(GOSRC)/relay/plugins/traffic/logging/main/main.go
	go build -buildmode=plugin -o $(DIST_PATH)/plugins/traffic/active/monitor.so $(GOSRC)/relay/plugins/traffic/monitor/main/main.go
	go build -o $(DIST_PATH)/relay $(GOSRC)/relay/main/main.go

test:
	go test relay/...

clean:
	rm -rf $(DIST_PATH)/*
	rm -rf $(GOPKG)/*
