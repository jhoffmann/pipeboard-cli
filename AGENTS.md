# Agent Guidelines for Pipeboard

## Build/Test Commands

- **Build**: `mise build` → outputs to `build/pipeboard`
- **Test**: `mise test` (runs `go test -tags=coverage ./... -cover`)
- **Single test**: `go test -run TestFunctionName ./package`
- **Format**: `mise fmt` (runs `go fmt ./...` and `prettier -lw .`) - **DO NOT RUN** (user will handle manually)
- **Lint/Vet**: `mise vet` (runs `go vet ./...`) - **DO NOT RUN** (user will handle manually)
- **Dev workflow**: `mise dev` (deps + format + analysis + test) - **DO NOT RUN** (user will handle manually)
- **Build output**: All build artifacts go in `build/` directory

## ⚠️ IMPORTANT: DO NOT RUN THE APPLICATION

- **NEVER run**: `mise run`, `go run .`, or execute the built binary
- This is a Terminal UI (TUI) application that will take over the terminal
- Running it will break terminal interaction and require force-killing the process
- Use build/test commands only for development and validation

## Code Style

- Use Go 1.25+ features, follow standard Go conventions
- Package imports: stdlib first, then external, then local (`pipeboard/...`)
- Struct fields: exported PascalCase, unexported camelCase
- Error handling: wrap with `fmt.Errorf("description: %w", err)`
- No comments unless documenting public APIs
- Use `context.Context` for cancellation/timeouts
- Prefer composition over inheritance

## Project Structure

- `main.go`: entry point and CLI setup
- `services/`: AWS API interactions and business logic
- `ui/`: Bubble Tea TUI components
- Module name: `pipeboard` (import as `pipeboard/services`, `pipeboard/ui`)
