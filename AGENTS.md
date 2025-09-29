# AGENTS.md - Development Guide for Coding Agents

## Build & Test Commands

- **Development workflow**: `mise dev` (runs dependencies, format, analysis, test)
- **Run application**: `mise run` or `go run cmd/pipeboard/*.go`
- **Build**: `mise build` (Linux) or `mise build-all` (all platforms)
- **Manual builds**: Use `go build -o build/pipeboard cmd/pipeboard/*.go` to output to build/ folder
- **Test**: `go test ./... -cover` or `mise test`
- **Single test**: `go test ./path/to/package -run TestName`
- **Coverage**: `mise coverage` (generates HTML report)
- **Format**: `go fmt ./...`
- **Lint/Analysis**: `go vet ./...`

## Build Output

- **IMPORTANT**: All build outputs must go into the `build/` directory, not the project root
- When building manually, always use: `go build -o build/pipeboard-cli cmd/pipeboard/*.go`

## Code Style & Conventions

- **Go version**: 1.23.0
- **Imports**: Standard library first, then third-party, then local packages with alias prefixes
- **Naming**: Use Go conventions (CamelCase for exports, camelCase for private)
- **Error handling**: Always wrap errors with context using `fmt.Errorf("description: %w", err)`
- **Types**: Define custom types for domain concepts (e.g., `Pipeline`, `ActionExecution`)
- **Comments**: Minimal comments, code should be self-documenting
- **Struct organization**: Group related fields, use composition over inheritance

## Project Structure

- `cmd/pipeboard/` - Main application entry point
- `internal/aws/` - AWS CodePipeline service integration
- `internal/ui/` - Bubble Tea terminal UI components
- Uses Cobra for CLI, Bubble Tea for TUI, AWS SDK v2
