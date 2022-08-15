FROM ubuntu:22.04 AS builder
RUN apt-get update && \
    apt-get install -y \
      build-essential \
      ca-certificates \
      golang \
      golang-1.18
ADD ./catcher ./catcher
ADD ./relay ./relay
ADD ./go.mod .
ADD ./go.sum .
ADD ./Makefile .
RUN set -ex && \
	make

FROM ubuntu:22.04
RUN apt-get update && \
    apt-get install -y \
      ca-certificates
COPY --from=builder /dist /dist
ENTRYPOINT [ "/dist/relay" ]
