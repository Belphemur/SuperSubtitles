# Design Decision Template

Use this template when adding a new design decision. Each decision within a file follows this structure.

---

## \<Short Decision Title\>

**Decision**: One or two sentences describing **what** was decided.

**Rationale**:

- Bullet points explaining **why** this decision was made
- Focus on trade-offs, alternatives considered, and benefits
- Each point should be a clear, standalone reason

**Implementation**: Where and how the decision is realised in the codebase. Reference file paths, interfaces, method names, and key types. This section is encouraged — it anchors the "why" to the "how" and helps future contributors find the relevant code.

---

### Guidelines

- **One file per domain** — group related decisions in a single file (e.g., all cache decisions in `cache.md`)
- **Decision** = what was chosen. Keep it factual.
- **Rationale** = why it was chosen. Focus on trade-offs and benefits.
- **Implementation** = where it lives in the code. File paths, interfaces, method names are all welcome here.
- If a decision has notable architectural detail (e.g., Redis key design, Lua scripts), add a subsection between Rationale and Implementation.
- Keep each decision self-contained — a reader should understand it without reading the rest of the file.
