# Cross-source query improvements

## User journey and observable results

1. Open the saved two-database sales query or run it from the CLI. The query identifies its source databases and the result's consistency scope before reading rows. A source change during a paged read produces a clear failure, never a silently mixed result.
2. Download pages and join them. Progress reports per-source rows loaded, rows processed, and lookup requests completed, in flight, and pending. Counters are cumulative and never imply a total that is unknown.
3. Show exact sales totals and per-capita results in Go and JavaScript. Fractional amounts, large amounts, nulls, and division follow one documented numeric policy and shared acceptance fixtures.
4. Choose **Full result** or **Visible rows** in DataTug before running a query. Page a large browser result without holding it all in UI memory. Visible-row mode joins and enriches only pages requested by the user, retaining completed pages for back navigation. Aggregates explain why they require full processing. Cancel or close the result and remove temporary storage. A later browser session also removes abandoned temporary databases.
5. Run a large CLI result to a streaming format with bounded memory and preserved access checks. HTTP lookups retain bounded concurrency, backpressure, and observable failures.

## Implementation sequence

| Stage | Repository ownership | Deliverable | Acceptance |
| --- | --- | --- | --- |
| 1 | OpenVaultDB | Versioned/snapshot paging contract with expiry and explicit stale-source errors | Mutate a source between pages; the next page fails. Same-source pages remain consistent. |
| 2 | DALgo Go and JS | Shared numeric policy, streaming arithmetic parity, and lookup cancellation | Golden cases for fractions, large values, nulls, division, and failed concurrent requests. |
| 3 | DataTug web | Adopt paging contract; cumulative progress; user-switchable full/visible-row modes; recover abandoned IndexedDB runs; bounded retries, lookup reuse, and worker boundary | Real browser journey through 120,000 rows, switching modes, lazy first/next/back pages, lookup failure, source mutation, cancellation, and restart cleanup. |
| 4 | DataTug CLI | Stream output and row lookups with backpressure | Two SQLite sources and a large generated result finish with bounded memory and exact output. |
| 5 | Demo project | Update fixtures and instructions for both runtimes | Saved query matches Go, JS, CLI, and browser output. |

## Boundaries

- Keep the current flat equality join and bounded dimension limits. General spillable joins are a separate feature.
- Visible-row mode applies only when output pages can be computed independently: one flat join, no global aggregate or result ordering, and no cross-row operation that changes the meaning of an unseen page. Total rows may remain unknown until the source is exhausted.
- A source-level page contract does not imply one atomic snapshot across independent databases. Report source revisions and the consistency scope honestly.
- Direct browser reads rely on OVDB authorization; CLI reads also apply DataTug's leaf access policies. Test and document this difference.
- Retry only idempotent reads, with a finite attempt and time budget. Never publish partial totals after an error.
- Keep existing output formats compatible; add a streaming format or path for large CLI results.

## Landing order

Land provider contracts first (OpenVaultDB and DALgo), then web and CLI consumers, then the reproducible demo. Verify each exact remote head, CI, and WB cleanup receipt.
