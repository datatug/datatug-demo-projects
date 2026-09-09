# inGitDB database

An inGitDB database root (the layout `ingitdb://` expects: `.ingitdb/root-collections.yaml`
plus one directory per collection). Open it with:

```
datatug query run --db ingitdb://<path-to-this-directory> --from support-notes --no-policies
```

## Collections

- **support-notes** — 8 support-ticket notes keyed by record ID, each carrying a
  `CustomerId` field (2 records for customer 5); see the dataset definition at
  [`../../recordsets/support-notes.recordset.json`](../../recordsets/support-notes.recordset.json).

Generated with `dalgo2ingitdb`/`ingitdb-go` (the same libraries `datatug-cli`'s
`ingitdb://` backend and its own `query run` tests use), not hand-authored, so the
on-disk layout is exactly what the real driver produces and reads back.
