# RTSPanda — AI Agent Workflow Reference

Last updated: 2026-03-09

---

## Three-Tool System

| Tool       | Role                               | Reads                             | Writes               |
|------------|------------------------------------|-----------------------------------|----------------------|
| **Claude** | Architect, reviewer, planner       | All AI files + code               | AI/*.md              |
| **Cursor** | Primary implementer                | TODO.md, ARCHITECTURE.md, FEATURES/ | Source files    |
| **Aider**  | Bulk edits, refactors, grep-heavy  | Specific file targets             | Source files         |

---

## When to Use Each Tool

### Use Claude when:
- Starting a new phase or planning sprint
- A decision needs architectural review
- Something unexpected was discovered during implementation
- A feature needs a spec before Cursor touches it
- Writing or updating AI coordination files

### Use Cursor when:
- Implementing tasks from `TODO.md` marked "Ready for Cursor"
- Building UI components, pages, API handlers
- Scaffolding new files from scratch
- The task fits in one focused session

### Use Aider when:
- Renaming or moving things across many files
- Bulk SQL or schema changes
- Grep-driven refactors across the codebase
- Tasks marked "Ready for Aider" in TODO.md

---

## Standard Workflow

### Before starting any task:

1. Read `AI/TODO.md` — pick the next "Ready for Cursor" task
2. Read `AI/ARCHITECTURE.md` — confirm where the code lives
3. Check `AI/DECISIONS.md` — do not re-open locked decisions
4. If the task has a FEATURES/ doc, read that too

### After completing a task:

1. Move the task from "Ready for Cursor" → "Done" in `AI/TODO.md`
2. If you discovered something unexpected, add a note to `AI/HANDOFF.md`
3. If a new decision was made, add it to `AI/DECISIONS.md`

### When blocked:

- Note the blocker in `AI/HANDOFF.md` under "Blockers"
- Do not guess at architecture — surface the question to Claude first

---

## File Roles (Quick Reference)

| File                              | Purpose                                           | Who edits it |
|-----------------------------------|---------------------------------------------------|--------------|
| `AI/TODO.md`                      | Task queue, acceptance criteria, dependencies     | Claude       |
| `AI/ARCHITECTURE.md`              | Authoritative system design, module layout        | Claude       |
| `AI/DECISIONS.md`                 | Locked architecture decisions                     | Claude       |
| `AI/PROJECT_CONTEXT.md`           | Stack, constraints, coding standards              | Claude       |
| `AI/CURRENT_FOCUS.md`             | Current sprint, what not to touch                 | Claude       |
| `AI/HANDOFF.md`                   | Session notes, blockers, open questions           | All tools    |
| `AI/WORKFLOW.md`                  | This file — agent coordination reference          | Claude       |
| `AI/FEATURES/*.md`                | Detailed specs for complex features               | Claude       |
| `AI/UXDesign/*.md`                | Visual design briefs for Cursor's UI work         | Claude       |

---

## Ground Rules

1. **Never re-open a locked decision** without flagging it in HANDOFF.md and asking Claude
2. **Never implement features not in TODO.md** — scope is controlled there
3. **No auth in Phase 1** — do not add it, even as a stub with logic
4. **mediamtx is a subprocess**, not a container — do not Dockerize it separately
5. **SQLite only** — do not introduce any other database
6. **sourceOnDemand: true** on all mediamtx paths — streams must close when unwatched
7. **No FFmpeg** unless absolutely unavoidable (flag it first)
