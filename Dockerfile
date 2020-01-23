
FROM golang:alpine AS builder
ADD ./go/src ./go/src
ADD ./Makefile .
RUN apk add --update make build-base
RUN set -ex && \
	make prep && \
	make compile

FROM alpine
RUN apk add --update ca-certificates
COPY --from=builder /go/dist /dist
ENTRYPOINT [ "/dist/relay" ]
