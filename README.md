# pulse
PulseAudio client implementation in pure Go.

Based on [github.com/yobert/pulse](https://github.com/yobert/pulse), which provided a very useful starting point.

Uses the pulseaudio native protocol to play audio without any CGO. The `proto` package exposes a very low-level API while the (currently very limited) `pulse` package is more convenient to use.

# status

- `proto` supports almost all of the protocol, shm support is still missing.

- `pulse` can do basic playback and recording.
