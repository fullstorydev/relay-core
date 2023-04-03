FROM alpine:3.16 AS builder
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

FROM alpine:3.16
RUN apk add --no-cache --update \
  ca-certificates
COPY --from=builder /dist /dist
COPY relay.yaml /etc/relay/relay.yaml
ENTRYPOINT [ "/dist/relay" ]
CMD [ "--config", "/etc/relay/relay.yaml" ]
