# ScaleLab starter

This module intentionally compiles while the core task remains incomplete. The learner has a CLI, data model and named stages, not a completed solution. Start with the parent `../README.md`; keep the baseline and add meaningful correctness tests before optimization.

```sh
go test ./...
go vet ./...
go run .
```

`aggregateBaseline` is a working but allocation-heavy baseline. `aggregateOptimized` is the unfinished implementation. A passing compile-smoke test does not satisfy the project. Your evidence package must include baseline/after raw benchmark output, environment, profile interpretation, capacity/SLO calculation, edge/fault analysis and limitations.
