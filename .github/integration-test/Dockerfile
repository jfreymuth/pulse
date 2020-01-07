FROM golang:1-alpine

RUN apk add --no-cache pulseaudio
RUN echo "load-module module-null-sink sink_name=DummySink" >> /etc/pulse/default.pa \
  && echo "load-module module-loopback sink=DummySink" >> /etc/pulse/default.pa

RUN echo $'#!/bin/sh\n\
pulseaudio --daemonize \
  --fail \
  --verbose \
  --system \
  --disallow-exit\n\
exec $@' > /entrypoint.sh \
  && chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]

ENV CGO_ENABLED=0
ENV PULSE_SERVER=unix:/run/pulse/native
ENV PULSE_COOKIE=/run/pulse/.config/pulse/cookie

RUN mkdir -p ${GOPATH}/src/github.com/jfreymuth/pulse
COPY . ${GOPATH}/src/github.com/jfreymuth/pulse/
WORKDIR ${GOPATH}/src/github.com/jfreymuth/pulse

RUN go mod download
RUN go build ./...
