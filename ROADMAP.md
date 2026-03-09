# go-stt Roadmap

OpenAI-compatible STT client for Go. Covers both HTTP batch and WebSocket streaming.

## Phase 1 — Extended HTTP API (v0.2) ✗

Expose full ox-whisper `/v1/audio/transcriptions` capabilities.

- [ ] **Verbose response** — word-level timestamps, confidence, segments
- [ ] **Output formats** — `text`, `srt`, `vtt` via `TranscribeRaw()` returning `[]byte`
- [ ] **Advanced options** — `WithPunctuate`, `WithSmartFormat`, `WithDiarize(speakers)`, `WithKeywords`, `WithPII`
- [ ] **`TranscribeReader`** — accept `io.Reader` instead of file path (streaming upload)
- [ ] **Model listing** — `Models()` via `GET /v1/models`

## Phase 2 — WebSocket Streaming (v0.3) ✗

Real-time streaming client matching ox-whisper `/v1/listen` protocol.

- [ ] **`StreamClient`** — persistent WebSocket connection with gorilla/websocket
- [ ] **Event callbacks** — `OnMetadata`, `OnResult`, `OnSpeechStarted`, `OnError`
- [ ] **Control messages** — `Finalize()`, `Close()`, `KeepAlive()`
- [ ] **PCM encoding** — s16le (default) and f32le binary frame sending
- [ ] **Connection params** — language, vad, interim_results, smart_format, punctuate, encoding, sample_rate
- [ ] **Auto-reconnect** — configurable retry with exponential backoff (Deepgram pattern)

Reference: Deepgram Go SDK `pkg/client/common/v1/websocket.go` — separate `Connect()` vs `AttemptReconnect()`, distinct close levels (stream vs protocol vs fatal).

## Phase 3 — Reliability (v0.4) ✗

Production hardening.

- [ ] **Retry with backoff** — configurable for transient HTTP errors (429, 502-504)
- [ ] **Circuit breaker** — `WithCircuitBreaker(maxFails, cooldown)` to avoid hammering dead service
- [ ] **Typed errors** — `STTError{StatusCode, Message}` with `errors.Is` support
- [ ] **Request timeout per-call** — context deadline override
- [ ] **Concurrent limiter** — `WithMaxConcurrency(n)` semaphore for batch workloads

Reference: AssemblyAI SDK uses `cenkalti/backoff` for polling; Deepgram separates fatal vs transient error paths.

## Phase 4 — Streaming Helpers (v0.5) ✗

Higher-level APIs for common streaming patterns.

- [ ] **`StreamFile`** — stream a file over WebSocket in configurable chunks (100ms default)
- [ ] **`StreamMicrophone`** — `io.Reader` adapter for real-time audio input
- [ ] **Interim coalescing** — buffer interim results, emit only when stable (reduces callback noise)
- [ ] **Utterance assembly** — collect `speech_final` segments into complete utterances
- [ ] **Channel-based API** — `Results() <-chan Result` as alternative to callbacks

Reference: Deepgram SDK exposes both callback and channel patterns.

## Phase 5 — Ecosystem Integration (v0.6) ✗

Convenience for consumers (dozor, vaelor, go-hully).

- [ ] **Telegram voice helper** — download voice/video-note → temp file → transcribe → cleanup
- [ ] **Audio format detection** — detect ogg/wav/mp3/flac from headers, pass appropriate encoding
- [ ] **Large file chunking** — split files >25MB by silence (VAD) for APIs with size limits
- [ ] **SRT/VTT parser** — parse subtitle output back into typed structs

## Non-Goals

- Audio recording/capture (OS-specific, out of scope)
- Model management (server-side concern)
- TTS (separate domain)

## Competitive Landscape

| Feature | go-stt | Deepgram SDK | AssemblyAI SDK | OpenAI SDK |
|---------|--------|-------------|---------------|------------|
| HTTP batch | v0.1 ✓ | ✓ | ✓ | ✓ |
| WebSocket streaming | v0.3 | ✓ (rich) | ✗ | ✗ |
| Word timestamps | v0.2 | ✓ | ✓ | ✓ |
| SRT/VTT output | v0.2 | ✗ | ✓ | ✓ |
| Auto-reconnect | v0.3 | ✓ | n/a | n/a |
| Retry/backoff | v0.4 | partial | ✓ | ✗ |
| Circuit breaker | v0.4 | ✗ | ✗ | ✗ |
| Typed errors | v0.4 | ✗ | ✓ | ✓ |
| Diarization | v0.2 | ✓ | ✓ | ✓ |
| PII redaction | v0.2 | ✓ | ✓ | ✗ |
| Telegram helper | v0.6 | ✗ | ✗ | ✗ |
| Zero dependencies | v0.1 ✓ | ✗ (gorilla) | ✗ (backoff) | ✗ |

**Differentiator**: go-stt targets self-hosted ox-whisper (CPU-only, zero data retention) while being compatible with any OpenAI-like STT API. Phases 2-3 add Deepgram-level streaming without Deepgram's cloud lock-in. Phase 5 adds opinionated helpers specific to our agent ecosystem.
