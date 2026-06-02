<p align="center">
  <img src="./imgs/openocta_logo.png" alt="OpenOcta" width="420">
</p>

<p align="center">
  <b>OpenOcta</b> — An open‑source, enterprise‑grade AI agent: one binary, deploy anywhere
</p>

> English | [简体中文](README.md)


OpenOcta takes design cues from [OpenClaw](https://github.com/openocta/openocta) and is **fully reimplemented** as a **single Go backend binary with an embedded frontend**, suited to in‑enterprise chat, workflow automation, and integrating business systems, APIs, and toolchains.

---

## Project overview

- **Positioning**: Provides a **unified AI Agent runtime and control plane** inside the enterprise, connecting business systems, HTTP/APIs, MCP tools, and custom skills for conversational workflows, task orchestration, and automation.
- **Form factor**: One binary that ships Gateway, Agent, Channels, Cron, and the embedded Control UI static assets; integrate via CLI, HTTP/WebSocket, and Webhooks.
- **Protocol compatibility**: Compatible with the OpenClaw Gateway WebSocket protocol and official Control UI behavior, so you can migrate from or run alongside OpenClaw.

---

## Community & feedback

- **Discussions & experience sharing**: Join the OpenOcta group to share deployment practices, integration scenarios, and best practices.
- **Issues & feature requests**: Ask in the group or open Issues / PRs in this repository.

<p align="center">
  <img src="./imgs/wechat.png" alt="OpenOcta community QR code" width="220" height="220">
  <br/>
  <sub>Scan to join the OpenOcta discussion group for updates and support</sub>
</p>

---

## Quick start

### Requirements

- **Go 1.24+** (build the backend)
- **Node ≥18** (frontend build only; not needed in production)
- Environment variable **`ANTHROPIC_API_KEY`** (required when using the `agent` command)

### Build and run the Gateway

```bash
# Build (Makefile recommended)
make build

# Or use the helper script
./build.sh build   # supports: ui | embed | go | build | clean | snapshot | release | docker

# Start the Gateway
make run
# or
./openocta gateway run
```

By default the Gateway listens on `http://127.0.0.1:18900`; HTTP and WebSocket share the same port. The **frontend is embedded with `go:embed`**, so open that URL in a browser to use the Control UI.

### Dev mode (frontend hot reload)

```bash
# Terminal 1: Gateway (build once first)
./openocta gateway run

# Terminal 2: frontend dev server (port 5173)
make run-ui
```

Open `http://localhost:5173` to debug the frontend with hot reload.

### Agent CLI

```bash
export ANTHROPIC_API_KEY=your-key
./openocta agent -m "Hello, echo test"
```

### Configuration

On first run, if no config file exists, OpenOcta seeds one from the embedded `openocta.json.example` into:

| Platform      | Default path                       |
|---------------|-------------------------------------|
| Linux / macOS | `~/.openocta/openocta.json`        |
| Windows       | `%APPDATA%\openocta\openocta.json` |

### macOS desktop install (.dmg)

- **`.dmg`**: Mounts a **read-only** volume under **`/Volumes/...`**. Drag the app to **Applications** or use the in-app install prompt so it lands at **`/Applications/OpenOcta.app`**. Do not treat `/Volumes/...` as the installed copy—**eject** the disk image in Finder. See `deploy/dist-README.md`.

---

## Documentation & references

### Language map

- **Chinese**
  - Root: `README.md`
  - Backend: `src/README.md`
  - Frontend: `ui/README.md`
- **English**
  - Root: `README.en.md` (this file)

### Main docs

- **Backend overview**: `src/README.md` — modules, migration status, and backend doc index.
- **Frontend**: `ui/README.md` — Control UI features, dev scripts, and layout.
- **Configuration & capabilities** (`src/docs/`):
  - `configuration.md` — overview: agents, channels, gateway, cron, hooks, memory, and more.
  - `mcp-configuration.md` — MCP server declarations, permissions, timeouts.
  - `request-tracing.md` — request tracing, token usage, and call chains.
  - `webhooks.md` — `/hooks/wake`, `/hooks/agent`, `/hooks/alert` payloads and use cases.
  - `architecture.md` — layered design on agentsdk-go and extension points.
  - `skills.md`, `tools.md`, `tools-builtin.md`, `tools-openocta.md` — skills and tools overview.

Upstream:

- [OpenClaw repository](https://github.com/openclaw/openclaw)
- [docs.openclaw.ai](https://docs.openclaw.ai) — official Gateway and configuration docs

---

## Repository layout

```text
OpenOcta/
├── src/                    # Go backend (Gateway, Agent, Channels, Cron, etc.)
│   ├── cmd/openocta/       # CLI entry and subcommands
│   ├── embed/              # Embedded assets (frontend, config-schema, openocta.json.example)
│   │   └── frontend/       # Built frontend (from build)
│   ├── pkg/                # Core packages
│   └── README.md           # Backend documentation
├── ui/                     # Control UI (Lit + Vite, WebSocket control plane)
│   └── README.md           # Frontend documentation
├── deploy/                 # Dockerfile, systemd service, and related deploy files
└── docs/                   # Docs and migration notes
```

- **Backend**: Go 1.24+, Gateway HTTP + WebSocket, Agent, Channels, Cron, config, and related services.
- **Frontend**: Control UI is **embedded via `go:embed`** in the single binary; no separate frontend deployment in production.
- **Single binary**: The built `openocta` ships the UI and config templates and can be distributed and run as‑is.

---

## License

This project is released under **GPLv3** with additional terms.

You may create derivative works from OpenOcta’s source, provided you:

- Do **not** replace or alter the OpenOcta logo or copyright notices.
- Keep derivative works compliant with GPLv3 obligations.

For commercial licensing, contact: **sales@aspirecn.com**.
