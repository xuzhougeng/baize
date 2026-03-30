# Repository Guidelines

## Project Structure & Module Organization
`cmd/myclaw` is the CLI entry point; `cmd/myclaw-desktop` hosts the Wails desktop app and checked-in web assets under `frontend/dist/`. Shared business logic lives in `internal/` packages such as `ai`, `app`, `knowledge`, `reminder`, `terminal`, and `weixin`. Keep new code in `internal/<domain>` unless it is an executable entry point. Repo docs and images live in `docs/`; release packaging scripts live in `scripts/` and `packaging/windows/`.

## Build, Test, and Development Commands
Use `go run ./cmd/myclaw` for the terminal app and `go run ./cmd/myclaw-desktop` for the desktop shell. `make dev` starts the desktop app in HTTP dev mode on `127.0.0.1:3415`. `make test` runs `go test ./...` across the repository. `make build-current` builds the CLI into `dist/`; `make package-linux`, `make package-windows`, and `make package-macos` create release archives.

## Coding Style & Naming Conventions
Target Go 1.24 and let `gofmt` own formatting; do not hand-align whitespace. Follow Go naming: exported identifiers use PascalCase, internal helpers use camelCase, package directories stay lowercase, and platform files use suffixes like `_windows.go` or `_stub.go`. Keep functions small and package boundaries clear; prefer extending existing `internal/*` packages over adding cross-package shortcuts.

## Tool Units
Reusable tool-style modules must be designed as self-descriptive units, so they can be reused by other AI projects without reverse-engineering app-specific code paths.

When adding a new tool unit:
- Put the executable logic in its own `internal/<tool>` package instead of burying it inside a transport layer such as WeChat, terminal, or desktop UI.
- Expose a stable tool name, a short purpose statement, a machine-oriented input contract, a machine-oriented output contract, and a human-readable help/usage text.
- Keep the phases separated: intent recognition decides whether the tool should run and prepares tool input; the tool package normalizes input and executes; the transport layer only renders results or delivers side effects.
- If the tool also has a shortcut command such as `/find`, treat that command as a thin registration layer over the tool unit. Register the shortcut in the runtime that actually owns it, not globally by default, and route `help` back to the tool's own usage text from that runtime.
- Update the registry below whenever a reusable tool is added, renamed, or removed.

### Registered Tool Units
- `everything_file_search`
  Package: `internal/filesearch`
  Purpose: Search local Windows files via Everything (`es.exe`) using either native queries or structured semantic filters.
  Input contract: `query`, `keywords`, `drives`, `known_folders`, `paths`, `extensions`, `date_field`, `date_value`, `limit`.
  Output contract: executed query, effective limit, result count, ordered file items with `index`, `name`, and `path`.
  Shortcut registration: `/find` and `/find help`, registered by the WeChat bridge at runtime after bridge startup.
  Current pipeline split: intent recognition in `internal/ai` and `internal/app`; tool execution in `internal/filesearch`; WeChat file delivery in `internal/weixin/media.go`.

## Testing Guidelines
Place tests beside the code they cover as `*_test.go`; this repo already follows that pattern in `internal/*` and `cmd/myclaw-desktop`. Prefer table-driven tests for routing, storage, and parser behavior. There is no stated coverage gate, but new logic should include focused tests and `make test` should pass before a PR is opened.

## Commit & Pull Request Guidelines
Install hooks with `make install-hooks`; `.githooks/commit-msg` enforces conventional subjects in the form `feat(scope): summary`, `docs(scope): summary`, or `chore(scope): summary`. Use lowercase scopes such as `model`, `desktop-chat`, or `ci`, matching recent history. PRs should explain user-visible behavior, list validation steps, link related issues, and include screenshots when desktop UI or docs output changes.
