export PROJECT_HOME := $(shell pwd)
export DIST_PATH     := $(PROJECT_HOME)/dist
export RELAY_MODULE := github.com/fullstorydev/relay-core

.PHONY: all compile test clean

all: compile

compile:
	go version
	go build -o $(DIST_PATH)/relay $(RELAY_MODULE)/relay/main
	go build -o $(DIST_PATH)/catcher $(RELAY_MODULE)/catcher/main

test: compile
	go test -v $(RELAY_MODULE)/...

clean:
	rm -rf $(DIST_PATH)/*
