FROM golang:1.22.3-alpine3.19 AS builder
RUN apk add --no-cache --update \
  alpine-sdk \
  ca-certificates \
  go
ADD ./catcher ./catcher
ADD ./relay ./relay
ADD ./go.mod .
ADD ./go.sum .
ADD ./Makefile .
RUN set -ex && \
	make

FROM alpine:3.19
RUN apk add --no-cache --update \
  ca-certificates
COPY --from=builder /dist /dist
COPY relay.yaml /etc/relay/relay.yaml
ENTRYPOINT [ "/dist/relay" ]
CMD [ "--config", "/etc/relay/relay.yaml" ]
