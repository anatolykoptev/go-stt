# go-stt

[![Go Reference](https://pkg.go.dev/badge/github.com/anatolykoptev/go-stt.svg)](https://pkg.go.dev/github.com/anatolykoptev/go-stt)
[![preflight](https://github.com/anatolykoptev/go-stt/actions/workflows/preflight.yml/badge.svg)](https://github.com/anatolykoptev/go-stt/actions/workflows/preflight.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

A Go client for speech-to-text servers that speak the OpenAI audio API: batch transcription over
`POST /v1/audio/transcriptions`, model listing over `GET /v1/models`, and real-time streaming over a
Deepgram-style WebSocket at `/v1/listen`. It works against a self-hosted whisper-style server, the
OpenAI API and other cloud providers that expose the same routes. Retry with backoff and a circuit
breaker are built in.

```sh
go get github.com/anatolykoptev/go-stt@latest
```

## Batch transcription

```go
client := stt.New("https://stt.example.com",
    stt.WithLanguage("en"),
    stt.WithAPIKey(os.Getenv("STT_API_KEY")), // sent as "Authorization: Bearer ..."; optional for self-hosted servers
    stt.WithRetry(3, 500*time.Millisecond),   // transient errors only: HTTP 429, 502, 503, 504
    stt.WithCircuitBreaker(5, 30*time.Second),
)

res, err := client.Transcribe(ctx, "voice.ogg") // *stt.Response{Text, Language, Duration}
```

| Method | Returns |
|---|---|
| `Transcribe`, `TranscribeReader(ctx, r, filename)` | `*Response` (text, language, duration) |
| `TranscribeVerbose` | `*VerboseResponse` with segments, word timings, speakers and utterances |
| `TranscribeRaw` | the server's body as `[]byte`, for `text`, `srt` or `vtt` output |
| `TranscribeURL`, `TranscribeURLVerbose` | download an audio URL to a temp file (`WithTempDir`), transcribe, delete it |
| `Models`, `IsAvailable` | model list; reachability (`/health`, then `/v1/models`) |

Options include `WithModel` (default `moonshine-v2`), `WithFormat`, `WithTimeout` (default 60 s),
`WithPunctuate`, `WithSmartFormat`, `WithDiarize`, `WithDiarizeSpeakers`, `WithKeywords`,
`WithCustomSpelling`, `WithRetryWithMaxDelay`. The default language is `ru`. `WithRetry` panics on
`maxAttempts` outside 1..100 or a non-positive delay, at construction time. Failures from the
server are `*stt.Error` with a `StatusCode` and `IsTransient()`; use `errors.As`.

## Streaming

```go
cs, err := stt.StreamWithChannels(ctx, "https://stt.example.com", stt.StreamParams{
    Language: "en", InterimResults: true, SampleRate: 16000, // encoding defaults to pcm_s16le
})
go func() { for ev := range cs.Events() { fmt.Println(ev.Transcript()) } }()
cs.Send(pcmFrame)   // binary PCM frames
cs.Finalize()       // flush the final result
cs.Close()          // graceful, bounded by a 5 s timeout; ForceClose() drops the connection
```

`StreamClient` with a `StreamHandler` is the callback form; `StreamFile` streams a PCM file in paced
chunks and returns the final events. `StreamParams.VAD` and `Punctuate` are `*bool` that default to
true when nil (use `stt.Bool(false)`). The events channel holds 64 events and drops on overflow;
`DroppedEvents()` counts them. Inbound WebSocket messages are capped at 10 MiB. `https` base URLs
become `wss`, `http` becomes `ws`; a base URL needs an explicit scheme.

## Status

Pre-1.0 (`v0.4.x`): the API can change in a minor release. Releases and the changelog come from
release-please, see [CHANGELOG.md](CHANGELOG.md). The planned work is in [ROADMAP.md](ROADMAP.md).
Contributions: [CONTRIBUTING.md](CONTRIBUTING.md). Every document is indexed in
[docs/INDEX.md](docs/INDEX.md). Report vulnerabilities as described in [SECURITY.md](SECURITY.md).

## Licence

Apache-2.0, see [LICENSE](LICENSE).
