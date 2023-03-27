export PROJECT_HOME := $(shell pwd)
export DIST_PATH     := $(PROJECT_HOME)/dist
export RELAY_MODULE := github.com/fullstorydev/relay-core

SHELL := /bin/sh
CLOUD_REGION_NAME := $(if $(CLOUD_REGION_NAME),$(CLOUD_REGION_NAME),us-west-2)
SERVICE_NAME := tokyo-bolombolo
ENV := 	$(if $(ENV),$(ENV),prod)
VERSION := $(if $(VERSION),$(VERSION),v-0.3.0)

.PHONY: all compile test clean

all: compile

bump-major: # @HELP Bumps the major version number.
bump-major:
	@bump2version --allow-dirty major

bump-minor: # @HELP Bumps the minor version number.
bump-minor:
	@bump2version --allow-dirty minor

bump-patch: # @HELP Bumps the patch version number.
bump-patch:
	@bump2version --allow-dirty patch

compile:
	go build -o $(DIST_PATH)/relay $(RELAY_MODULE)/relay/main
	go build -o $(DIST_PATH)/catcher $(RELAY_MODULE)/catcher/main

clean:
	rm -rf $(DIST_PATH)/*

deploy-service:
	aws --region $(CLOUD_REGION_NAME) cloudformation deploy --template-file template.yml --stack-name bolombolo-$(ENV) --parameter-overrides BearingEnv=$(ENV) Tag=$(VERSION) --capabilities CAPABILITY_NAMED_IAM

test: compile
	go test -v $(RELAY_MODULE)/...

update-image:
	aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin 757917910993.dkr.ecr.us-west-2.amazonaws.com \
	&& docker buildx build --platform=linux/amd64 -t $(SERVICE_NAME) . \
	&& docker tag $(SERVICE_NAME):latest 757917910993.dkr.ecr.us-west-2.amazonaws.com/$(SERVICE_NAME):$(VERSION) \
	&& docker push 757917910993.dkr.ecr.us-west-2.amazonaws.com/$(SERVICE_NAME):$(VERSION)

