export PROJECT_HOME := $(shell pwd)
export DIST_PATH     := $(PROJECT_HOME)/dist
export PLUGIN_DIST_PATH := $(DIST_PATH)/plugins/traffic
export RELAY_MODULE := github.com/fullstorydev/relay-core

.PHONY: all plugins cli compile test clean

all: compile

plugins:
	mkdir -p $(DIST_PATH)/plugins/traffic/active
	mkdir -p $(DIST_PATH)/plugins/traffic/inactive
	go build -buildmode=plugin -o $(PLUGIN_DIST_PATH)/active/010-paths.so $(RELAY_MODULE)/relay/plugins/traffic/paths-plugin/main
	go build -buildmode=plugin -o $(PLUGIN_DIST_PATH)/active/015-paths.so $(RELAY_MODULE)/relay/plugins/traffic/content-blocker-plugin/main
	go build -buildmode=plugin -o $(PLUGIN_DIST_PATH)/active/020-relay.so $(RELAY_MODULE)/relay/plugins/traffic/relay-plugin/main
	go build -buildmode=plugin -o $(PLUGIN_DIST_PATH)/active/030-logging.so $(RELAY_MODULE)/relay/plugins/traffic/logging-plugin/main

cli:
	go build -o $(DIST_PATH)/relay $(RELAY_MODULE)/relay/main
	go build -o $(DIST_PATH)/catcher $(RELAY_MODULE)/catcher/main

compile: plugins cli

test: compile
	go test -v $(RELAY_MODULE)/...

clean:
	rm -rf $(DIST_PATH)/*
