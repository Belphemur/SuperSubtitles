---
name: run-lint
description: "Run golangci-lint for the SuperSubtitles repository. Use when asked to lint, check style, validate CI lint expectations, or investigate lint failures in Go code."
argument-hint: "Optional package path or golangci-lint arguments, for example: ./internal/grpc or --fix"
---

# Run Lint

Use this skill for linting and lint-related validation in this repository.

## When To Use

- The user asks to run lint
- You need to verify a code change against CI lint requirements
- You need to inspect or summarize `golangci-lint` failures

## Procedure

1. Run commands from the repository root.
2. Default to `golangci-lint run` unless the user asks for a narrower scope.
3. If the user provides a package path or extra flags, append them to the lint command.
4. Report the failures with file paths and the relevant linter names.
5. If the request includes fixing code, run formatting or targeted edits first, then rerun lint.

## Repository Notes

- CI expects `golangci-lint run` with the repository `.golangci.yml`.
- A clean lint pass is required before committing.
- Pair lint validation with `go vet ./...`, `go test -race ./...`, and `go build ./...` when finishing substantial changes.
