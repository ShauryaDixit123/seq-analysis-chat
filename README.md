# ATGC

A web application for comparing DNA/RNA sequences from FASTA files. Upload sequences through a chat-style UI, run pairwise analysis (dynamic programming or dot product), and explore results as interactive scatter plots.

The module name **atgc** refers to the four nucleotide bases: adenine (A), thymine (T), guanine (G), and cytosine (C).

## Features

- **Chat interface** — Send messages and attach one or more FASTA files per session.
- **FASTA parsing** — Supports `.fa`, `.fasta`, `.fna`, `.ffn`, `.faa`, and `.frn`. Sequences are normalized (uppercase, whitespace stripped) and validated against allowed bases (`A`, `C`, `G`, `T`, `U`, `N`, `-`, `.`, `*`).
- **Sequence analysis modes** (selected in the UI):
  - **Dynamic programming** — Builds a score table over character pairs (match extends prior score; mismatch takes max of neighbors).
  - **Dot product** — Per-position matrix: `1` when bases match, `0` otherwise.
  - **Global / local alignment** — Exposed in the UI; currently routed through the same dynamic-programming implementation on the server.
- **Pairwise comparisons** — With two or more FASTA files in a session, all unique pairs are compared in parallel (goroutines).
- **Visualization** — Results render as Plotly scatter plots (row × column × score) in the chat, with a picker when multiple pair comparisons exist.
- **Session persistence** — Chat sessions and messages are stored in memory on the server; the browser keeps `session_id` in `localStorage`.

## Tech stack

| Layer      | Technology                          |
|-----------|--------------------------------------|
| Backend   | Go 1.25, [Gin](https://github.com/gin-gonic/gin) |
| Frontend  | HTML/CSS/vanilla JS, [Plotly.js](https://plotly.com/javascript/) (CDN) |
| Assets    | `embed` — templates and static files baked into the binary |

## Project layout

```
.
├── main.go                 # HTTP server, routes, embedded assets
├── go.mod
├── src/
│   ├── methods/
│   │   ├── dynamic.go      # Dynamic programming matrix
│   │   ├── dot.go          # Dot-product match matrix
│   │   ├── struct.go       # Method type
│   │   ├── fasta/
│   │   │   └── parse.go    # FASTA reader
│   │   └── handlers/
│   │       ├── chat.go     # Chat UI + upload + analysis API
│   │       └── index.go    # Standalone DP JSON endpoint
│   └── types/
│       ├── chat.go         # Chat/session/attachment models
│       ├── ctx.go          # App context (Gin + context.Context)
│       └── dynamic.go      # MethodRequestBody
├── templates/
│   └── chat.html
└── static/
    ├── css/chat.css
    └── js/chat.js
```

## Requirements

- Go **1.25** or newer
- Network access for Plotly CDN (charts only; the app shell works offline)

## Quick start

```bash
# From the repository root
go run .

# Server listens on http://localhost:8080 (Gin default)
```

Open [http://localhost:8080](http://localhost:8080) in a browser.

### Typical workflow

1. Open the chat page (`/`).
2. Choose an **Analysis** mode (e.g. Dynamic programming or Dot product).
3. Attach **at least two** FASTA files (you can upload one file first; the assistant will ask for more).
4. Send the message — the server parses sequences, runs all pairwise comparisons, and returns matrices attached to the assistant reply.
5. Use the comparison dropdown to switch between pair plots when multiple matrices are returned.

For chat-only messages (no analysis), set Analysis to **None (chat only)**.

## API reference

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check (`{"message":"OK"}`) |
| `GET` | `/` | Chat HTML page |
| `POST` | `/api/chat/session` | Create a new chat session |
| `GET` | `/api/chat/session/:session_id` | List messages for a session |
| `GET` | `/api/chat?session_id=…` | List messages (query param variant) |
| `POST` | `/api/chat?session_id=…&process_type=…` | Post message + optional FASTA uploads (`multipart/form-data`: `message`, `files`) |
| `POST` | `/dynamic` | JSON body `{ "sequence1", "sequence2", "plot" }` — dynamic programming (handler currently binds input; extend for full JSON response) |

### `process_type` query values

| Value | Behavior |
|-------|----------|
| `none` | Chat only; no matrix computation |
| `dynamic_programming` | DP score table |
| `dot_product` | Match / mismatch matrix |
| `global_alignment` | DP table (same code path as dynamic programming today) |
| `local_alignment` | DP table (same code path as dynamic programming today) |

## Algorithms (summary)

**Dot product** — For sequences of length *m* and *n*, produces an *m×n* matrix where cell `(i,j)` is `1` if `seq1[i] == seq2[j]`, else `0`.

**Dynamic programming** — Character arrays are compared with a recurrence: matching characters add to the score from the northwest cell; mismatches take the maximum of available neighbors (north, west, northwest). The full table is returned for visualization.

## Configuration & limits

- **Request timeout (client)** — 120 seconds for analysis requests (`static/js/chat.js`); large FASTA files may need smaller inputs or a higher timeout.
- **In-memory storage** — Sessions and messages are lost on server restart; there is no database.
- **Upload parsing** — Only the **first** record in each FASTA file is used (`fasta.First`).

## Development

```bash
# Run tests (if added)
go test ./...

# Build binary
go build -o atgc .
```

Vendor directory is gitignored; dependencies resolve via `go mod` on build.

## License

Not specified in the repository. Add a `LICENSE` file if you plan to distribute this project.
