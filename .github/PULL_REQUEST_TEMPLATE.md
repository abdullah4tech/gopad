## Summary

<!-- One or two sentences: what does this PR do and why? -->

## Type of change

- [ ] Bug fix
- [ ] Feature / enhancement
- [ ] Refactor (no behavior change)
- [ ] Docs / comments only
- [ ] Build / tooling

## Related issue

Closes #

## Changes

<!-- Bullet list of the key changes. Skip obvious ones — focus on the non-trivial decisions. -->

-
-

## Data-safety checklist

- [ ] No code path can lose or corrupt note content
- [ ] SQLite writes still go through WAL mode
- [ ] The debounce auto-save logic is untouched or intentionally modified (explain below if so)

## Testing

<!-- How did you verify this works? Include build steps if non-trivial. -->

```sh
make build
./gopad
```

- [ ] Opened app, created and edited a note — content persisted after restart
- [ ] Find/replace still works
- [ ] Multi-note switching still works

## Breaking changes

<!-- Anything that changes the DB schema, the JS↔Go bridge API, or the build requirements? -->

None / describe here.

## Screenshots (if UI change)

<!-- Before / after or a short screen recording. -->
