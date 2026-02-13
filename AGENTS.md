# Agents

## Go Lint After Edits

After editing `.go` files, run `golangci-lint` scoped to the packages you touched — not the entire project. This catches issues incrementally while they're small and easy to fix.

### Workflow

1. After making changes to `.go` files, identify the packages that were modified.
2. Run the linter scoped to those packages:

```bash
golangci-lint run ./path/to/changed/package/...
```

3. If there are findings, fix them before moving on to the next task.
4. If a finding is a false positive or intentional, note it to the user rather than silently adding a nolint directive.

### Guidelines

- **Scope narrowly**: Only lint the packages you changed. Don't run against the whole repo unless asked.
- **Fix before continuing**: Treat lint issues like compiler errors — resolve them before moving on.
- **Don't suppress blindly**: Never add `//nolint` directives without discussing with the user first.
- **Formatter included**: The project uses `gofumpt` via golangci-lint's formatter support. Formatting issues will surface in the output too.
