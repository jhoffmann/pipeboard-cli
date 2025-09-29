# pipeboard

```
░█▀█░▀█▀░█▀█░█▀▀░█▀▄░█▀█░█▀█░█▀▄░█▀▄
░█▀▀░░█░░█▀▀░█▀▀░█▀▄░█░█░█▀█░█▀▄░█░█
░▀░░░▀▀▀░▀░░░▀▀▀░▀░▀░▀▀▀░▀░▀░▀░▀░▀▀░
```

A terminal user interface for monitoring AWS CodePipelines.

## Features

- Monitor AWS CodePipeline status in a terminal UI
- Filter pipelines by name pattern
- Real-time pipeline status updates
- Structured logging with configurable levels

## Build

```bash
go build -o build/pipeboard-cli cmd/pipeboard/*.go
```

Or using mise-en-place:

```bash
mise build
```

## Run

```bash
./build/pipeboard-cli <filter>
```

Where `<filter>` is a pattern to match pipeline names.

## Requirements

- Go 1.24+
- AWS credentials configured (if you can run the AWS CLI, you should be good to go)

