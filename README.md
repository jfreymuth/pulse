# pulse
[![GoDoc](https://godocs.io/github.com/jfreymuth/pulse?status.svg)](https://godocs.io/github.com/jfreymuth/pulse)

PulseAudio client implementation in pure Go.

Based on [github.com/yobert/pulse](https://github.com/yobert/pulse), which provided a very useful starting point.

Uses the pulseaudio native protocol to play audio without any CGO. The `proto` package exposes a very low-level API while the `pulse` package is more convenient to use.

# status

- `proto` supports almost all of the protocol, shm support is still missing.

- `pulse` implements sufficient functionality for most audio playing/recording applications.

# examples

see [demo/play](demo/play/main.go) and [demo/record](demo/record/main.go)
