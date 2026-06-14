---
name: Crash / data loss
about: Notes lost, database corrupted, or app crashed unexpectedly
title: "crash: "
labels: "bug, data-loss, priority: high"
assignees: ""
---

## What was lost or corrupted

<!-- Describe exactly what data is missing or wrong. -->

## What triggered it

<!-- e.g. suspend/resume, kernel panic, power loss, closing the window, etc. -->

## Steps to reproduce (if known)

1.
2.
3.

## Database state

<!-- Run: sqlite3 ~/.local/share/gopad/notes.db ".tables" and paste the output. -->
<!-- If the file is missing, note that here. -->

```
<paste output here>
```

## Environment

- **OS / distro**:
- **WebKitGTK version**: <!-- `pkg-config --modversion webkit2gtk-4.1` -->
- **GoPad version / commit**:
- **Filesystem type**: <!-- ext4, btrfs, etc. -->
- **Was WAL mode active?**: <!-- `PRAGMA journal_mode;` in sqlite3 -->

## stderr / logs

<!-- Paste any output from the terminal you launched GoPad in. -->

```
<paste here>
```
