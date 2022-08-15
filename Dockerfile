
FROM golang:1.18-alpine3.16 AS builder
ADD ./catcher ./catcher
ADD ./relay ./relay
ADD ./go.mod .
ADD ./go.sum .
ADD ./Makefile .
RUN apk add --update make build-base git binutils-gold
RUN set -ex && \
    unset GOPATH && \
	make

FROM alpine
RUN apk add --update ca-certificates
COPY --from=builder /go/dist /dist
ENTRYPOINT [ "/dist/relay" ]
