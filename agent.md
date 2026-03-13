# SuperSubtitles Agent Profile

## Default Workflow

- Use the `parser-real-data-test` skill when the user provides a feliratok.eu URL and asks for parser tests from real data.
- Use the `run-lint` skill before committing or when lint validation is requested.
- Use the `record-decision` skill when parser/business logic changes introduce a non-trivial design decision.

## Parser Real-Data Requests

When the user gives a show URL:

1. Invoke the `parser-real-data-test` skill.
2. Create one dedicated real-data test file per show.
3. Fix parser business logic if the new test reveals a bug.
4. Re-run parser tests and lint checks before committing.
