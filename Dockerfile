
FROM golang:1.18-alpine3.16 AS builder
ADD ./go/src ./go/src
ADD ./Makefile .
RUN apk add --update make build-base git binutils-gold
RUN set -ex && \
	make

FROM alpine
RUN apk add --update ca-certificates
COPY --from=builder /go/dist /dist
ENTRYPOINT [ "/dist/relay" ]
