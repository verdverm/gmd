# `cmd/gmd/` File Organization

Each file contains exactly one cobra command (variable), except `main.go`
(entry point, `rootCmd`, and `init()` that registers all top-level commands).

## Naming convention

```
<command>_<subcmd>_<subsubcmd>.go
```

Files named as the command chain they implement, joined by underscores.

