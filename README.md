# datatug-demo-projects
A demo project for [DataTug.app](https://datatug.app)

It can be run at: https://datatug.app/pwa/agent/github/project/datatug-demo-project@datatug

## Folder: [`datatug`](demo-project-1)
The [datatug](demo-project-1) folder contains the definition of the DataTug project.

We store DataTug project definition in a sub-folder,
so it can be placed within your existing repository without any conflicts. 

## What this demo shows

One DataTug project, three data sources, tied together by semantics rather than by
copy-pasted IDs:

- **SQLite** — the classic [Chinook](https://github.com/lerocha/chinook-database) catalog
  (`Customer`, `Invoice`, `InvoiceLine`, `Track`, ...), described in
  [`dbmodels/chinook`](demo-project-1/dbmodels/chinook).
- **inGitDB** — a git-native `support-notes` collection
  ([`data/ingitdb`](demo-project-1/data/ingitdb)) keyed by `CustomerId`, standing in for a
  second, non-relational system of record about the same customers.
- **HTTP** — two public, keyless reference endpoints
  ([`queries/reference`](demo-project-1/queries/reference), with recorded snapshots under
  [`fixtures/http`](demo-project-1/fixtures/http) so the demo works offline): country facts
  by name, and a currency exchange rate by currency code.

`Customer`, `Invoice` and `Country` ([`entities`](demo-project-1/entities)) declare
`NamePatterns` so a `CustomerId`/`customer_id` column, an `InvoiceId` column or a `Country`
column can be recognized as the same semantic field wherever it appears. Three
parameterized queries ([`queries/customers`](demo-project-1/queries/customers),
[`queries/invoices`](demo-project-1/queries/invoices)) declare which entity field each
parameter binds to (`Parameters[].Meta`), and
[`policies`](demo-project-1/policies) shows row- and column-level access control: an
`admin` role sees everything, while a `support` role sees Canadian Customer, Invoice and
support-note rows under separate rules and never sees Customer email addresses. The
[Phase 1 acceptance fixture](demo-project-1/fixtures/chinook/README.md) pins the Chinook
database revision and the exact customer IDs and counts used by the demo journey.

## [License](LICENSE)
Licensed under [Creative Commons Zero v1.0 Universal](https://creativecommons.org/publicdomain/zero/1.0/)
