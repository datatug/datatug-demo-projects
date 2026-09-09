# Access policies

`customers.yaml` is a DALgo access policy document (`pkg/accesspolicies` in `datatug-cli`
loads every `*.yaml`/`*.yml`/`*.json` file here — see `datatug query run --policies-dir`).
It declares one principal policy set with two roles:

- **admin** — unrestricted read/write on every collection (`path: /**`).
- **support** — read-only on three independently declared resources: `Customer` rows where
  `Country = 'Canada'`, `Invoice` rows where `BillingCountry = 'Canada'`, and
  `support-notes` rows where the persisted `Country = 'Canada'`. `Email` is left out of
  the Customer field allow-list.

The separate rules are intentional. Access to a Customer row never implies access to its
Invoice or support-note rows; every related lookup must pass the policy for its own source.

Try it against the demo's SQLite chinook database (adjust `--db` to your local Chinook
copy — this repo does not commit the `.sqlite` file itself, see
`environments/local/servers/db`):

```
# admin: every customer, Email included
datatug query run --db sqlite://<path-to-chinook.sqlite> --from Customer \
  --policies-dir policies --as boss --role admin

# support: Canada-only, Email hidden
datatug query run --db sqlite://<path-to-chinook.sqlite> --from Customer \
  --policies-dir policies --as agent --role support
```
