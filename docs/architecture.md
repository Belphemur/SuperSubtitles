# SuperSubtitles — Architecture Documentation

This is the main index for SuperSubtitles architecture documentation.

## Documentation Structure

The architecture documentation is split into focused documents for easier reading and maintenance:

### [Overview](./overview.md)

High-level description of what the application does, its architecture diagram, and component relationships.

**Contents:**

- What the app does (7 main features)
- High-level architecture diagram with gRPC server
- Component relationships (gRPC Server, Client, Parser, Services, Models, Config)

### [gRPC API](./grpc-api.md)

Complete gRPC API documentation including proto definitions, endpoints, and usage examples.

**Contents:**

- Proto definition and code generation
- All 6 RPC methods with examples
- Data models and enums
- Error handling and testing
- Configuration and deployment
- Design decisions (no TLS, model conversion, reflection)

### [Data Flow](./data-flow.md)

Detailed explanation of data flow for all major operations.

**Contents:**

- Show list fetching
- Subtitle fetching with pagination
- Third-party ID extraction
- Recent subtitles fetching (main page with ID filtering)
- Subtitle download with episode extraction

### [Testing](./testing.md)

Testing infrastructure, strategies, and patterns.

**Contents:**

- HTML fixture generator (`testutil` package)
- Test strategy (no external frameworks, programmatic fixtures)
- Test coverage (parser, client, service tests)
- Running tests

### [Design Decisions](./design-decisions.md)

Key architectural and design decisions with rationale.

**Contents:**

- Partial failure resilience
- Generic parser interfaces
- Batch processing
- Programmatic test fixtures
- Parser reusability
- Parser handles all normalization (no separate converter)
- Client architecture considerations
- Parallel pagination
- Error handling strategy
- Pluggable cache with factory pattern (memory / Redis/Valkey)

### [Deployment](./deployment.md)

Configuration, CI/CD, dependencies, and deployment information.

**Contents:**

- Configuration (YAML, env vars)
- CI/CD pipeline (lint, test, build, release)
- Dependencies and version management
- Docker deployment
- Local development setup
- Monitoring and logging

## Quick Links

- **Getting Started**: See [Overview](./overview.md) for high-level understanding
- **Understanding Operations**: See [Data Flow](./data-flow.md) for how data moves through the system
- **Contributing**: See [Testing](./testing.md) for test patterns and [Design Decisions](./design-decisions.md) for architectural guidelines
- **Deploying**: See [Deployment](./deployment.md) for configuration and deployment options

## File Organization

```
docs/
├── architecture.md        # This file (index)
├── overview.md           # High-level architecture
├── grpc-api.md           # gRPC API documentation
├── data-flow.md          # Detailed operation flows
├── testing.md            # Testing infrastructure
├── design-decisions.md   # Architectural decisions
└── deployment.md         # Config, CI/CD, deployment
```
