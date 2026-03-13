---
name: parser-real-data-test
description: "Create parser regression tests from real feliratok.eu pages. Given a show URL, fetch HTML, build representative fixture-based parser tests in a new file, and fix parser business logic when tests fail."
argument-hint: "Show URL, for example: https://feliratok.eu/index.php?sid=13108"
---

# Parser Test From Real Data

Use this skill when the user gives a real feliratok.eu page URL and asks for parser coverage or bug fixes.

## When To Use

- User provides a feliratok.eu show URL and wants a parser test
- Parser behavior differs from real production HTML
- A regression needs to be captured with representative real-world rows
- A parser bug should be fixed together with a failing test

## Procedure

1. Validate the input URL and identify parser target:
- Subtitle listing pages (`sid=`) map to `internal/parser/subtitle_parser*.go`.
- Show list pages (`sorf=` or show listing pages) map to `internal/parser/show_parser*.go`.

2. Fetch page HTML content from the URL.

3. Derive representative test rows from real data:
- Include multiple languages if present.
- Include at least one edge case likely to break parsing.
- Keep the test focused; do not copy entire page HTML into tests.

4. Build fixture-based tests (no raw giant HTML literals):
- Use helpers in `internal/testutil/html_fixtures.go`.
- Prefer `GenerateSubtitleTableHTML` or `GenerateShowTableHTML` style helpers.

5. Create a new show-specific test file:
- Subtitle parser: `internal/parser/subtitle_parser_real_data_<show_slug>_test.go`
- Show parser: `internal/parser/show_parser_real_data_<show_slug>_test.go`
- Keep one show per file for readability and maintenance.

6. Assertions to include:
- Parsed show name and show ID
- Season/episode extraction (or season-pack metadata)
- Language code mapping where relevant
- Stable identity checks (subtitle ID / filename where available)

7. If tests fail, fix parser business logic in production code:
- Update regex/parsing branches in parser files
- Add/update focused unit tests for parsing helpers to prevent regressions
- Preserve existing behavior outside the bug scope

8. Validate changes:
- Run `go test ./internal/parser`
- If broader changes were made, run `/run-lint`

## Rules

- Do not hardcode full website HTML dumps in test files.
- Keep tests deterministic and representative, not exhaustive mirrors.
- Avoid changing unrelated parser behavior.
- Any parser logic change must be covered by tests.
