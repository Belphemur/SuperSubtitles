# Documentation Agent Guide

Apply these rules when editing files under `docs/`.

## Style

- Describe behavior, architecture, and rationale rather than copying implementation details.
- Keep design decision records specific: file paths, interfaces, and implementation details are appropriate there.
- Keep all other docs focused on domain concepts and user-visible behavior.
- Avoid repeating the same explanation across multiple docs.

## Update Targets

- Update `grpc-api.md` when request, response, or error behavior changes.
- Update `data-flow.md` when operation flows change.
- Update `configuration.md` for config or environment variable changes.
- Update `testing.md` when test strategy or fixtures change.
- Update the matching file under `design-decisions/` when a change affects architectural rationale.

## Validation

- Keep command examples accurate.
- Keep references aligned with the current repository structure.
- Prefer concise additions over large rewrites unless the structure is wrong.
