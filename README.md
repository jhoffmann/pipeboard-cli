# Pipeboard - AWS CodePipeline TUI Monitor

A Terminal User Interface (TUI) tool for monitoring AWS CodePipeline status using Go.

![demo](https://github.com/jhoffmann/pipeboard-cli/blob/main/demo/demo.gif?raw=true)

## Features

- 📋 List AWS CodePipelines with pagination
- 🔍 Filter pipelines by name (built-in search)
- 📊 View detailed pipeline stages and their status
- 🎨 Color-coded status indicators
- ⚡ Fast, responsive interface
- 🔄 Real-time refresh capabilities

## Prerequisites

- Go 1.19 or later
- AWS credentials configured (via AWS CLI, environment variables, or IAM roles)
- AWS region configured

## Installation

```bash
git clone <repository>
cd pipeboard-v2
go build -o pipeboard
```

## Usage

**A filter argument is required** to start the application.

```bash
./pipeboard <filter>
```

### Command Line Options

```bash
./pipeboard [options] <filter>
```

Options:

- `--log, -l`: Number of log lines to retrieve (default: 100)

Arguments:

- `filter`: Pipeline name filter pattern (required)

## Configuration

The application uses the AWS SDK's default credential chain:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. AWS credentials file (`~/.aws/credentials`)
3. IAM roles (when running on EC2)

Required AWS permissions:

- `codepipeline:ListPipelines`
- `codepipeline:ListPipelineExecutions`
- `codepipeline:ListActionExecutions`
- `codebuild:BatchGetBuilds`
- `logs:GetLogEvents`
- `logs:DescribeLogStreams`
