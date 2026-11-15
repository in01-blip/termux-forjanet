# termux-forjanet

**A native MCP (Model Context Protocol) server for Android, running inside
[Termux](https://termux.dev).** It lets any MCP-compatible AI client (Claude,
Claude Code, Gemini CLI, etc.) control an Android phone — clipboard, battery
and WiFi status, a sandboxed file area, and an optional allowlisted shell —
over stdio.

Written in Go, in Spanish (`README.es.md`, tool descriptions, log messages),
built for two real-world Android CPU architectures at once:

- **ARM64 / `android/arm64`** (aarch64) — the architecture of virtually every
  phone sold since ~2017.
- **ARM32 / `android/arm` (armeabi-v7a)** — still common on older and budget
  devices, where a lighter runtime profile avoids noticeable lag.

> 🇪🇸 Documentación completa en español: [`README.es.md`](README.es.md)

## Why this exists

This is an **independent project**. It reviews and takes inspiration from
three existing Termux/MCP projects — [TecnicalBot/termux-mcp](https://github.com/TecnicalBot/termux-mcp),
[hyhAsma/termux-api-tools-mcp-server](https://github.com/hyhAsma/termux-api-tools-mcp-server)
and [kahz12/DroidMCP](https://github.com/kahz12/DroidMCP) — but its code is
written from scratch, as a single-binary MCP server with a **default-deny
security model**: nothing sensitive (shell execution, filesystem access) is
exposed unless the user explicitly opts in with environment variables.

## Android-specific build reality (please read before opening an issue)

- **`android/arm64` cross-compiles in pure Go** (`CGO_ENABLED=0`) — no
  Android NDK required. CI builds and verifies this on every push.
- **`android/arm` (32-bit) always requires CGO**, whether you cross-compile
  from a PC or build natively inside Termux itself — confirmed by a real
  user run of `scripts/install.sh` on an armv7l device, which failed with
  `android/arm requires external (cgo) linking, but cgo is not enabled`
  even when building on the phone. CGO needs a C compiler targeting the
  destination's libc (Android's bionic). Building **inside Termux**, host
  and target are the same system, so no NDK is needed at all: installing
  `clang` (`pkg install -y clang`) is enough, since Termux already ships a
  clang that targets that exact architecture's bionic. `scripts/install.sh`
  installs `clang` and sets `CGO_ENABLED=1` automatically. The NDK is only
  needed if you cross-compile the 32-bit binary **from a PC** instead —
  there, host (glibc) and target (bionic) really are different systems.
- `GOOS=android` (not `linux`) is required for any build that will run
  inside Termux: Android kills a plain `linux` binary the first time it
  tries to run an external command.

## Quick start (on the Android device, inside Termux)

```bash
pkg install -y golang git
git clone <this-repo-url> termux-forjanet && cd termux-forjanet
bash scripts/install.sh      # detects arch + RAM, builds natively, sets a profile
source "$PREFIX/etc/termux-forjanet/profile.env"
termux-forjanet doctor
```

## Repository layout

```
cmd/termux-forjanet/       entry point + "doctor" diagnostic subcommand
internal/config/           env vars, active profile, build architecture
internal/server/           minimal MCP protocol over stdio (JSON-RPC), zero deps
internal/audit/            JSONL audit log with secret redaction
internal/tools/system/     battery, WiFi, profile info            [safe]
internal/tools/clipboard/  Android clipboard via termux-api        [safe]
internal/tools/files/      sandboxed read/write                    [sensitive, opt-in]
internal/tools/exec/       allowlisted shell execution              [dangerous, opt-in]
scripts/install.sh         architecture-aware installer (run INSIDE Termux)
scripts/cross-build-pc.sh  build from a PC (arm64 only, no NDK)
```

## Security model

- Default-deny: `files` and `exec` stay off until
  `TERMUXFORJANET_ENABLE_SENSIBLE=true` / `TERMUXFORJANET_ENABLE_PELIGROSO=true`.
- `exec` with no allowlist (`TERMUXFORJANET_SHELL_ALLOWLIST`) never registers —
  the tool doesn't exist on the wire if there's nothing to allow.
- `files` resolves symlinks and rejects any path that escapes
  `TERMUXFORJANET_ROOT`.
- Every call can be logged to a JSONL audit file with secret-looking fields
  (tokens, passwords, SMS bodies) redacted.

## License

MIT — see [LICENSE](LICENSE).
