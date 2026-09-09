# Entity: Country

## Fields

- **Exported**:
  [JSON](https://raw.githubusercontent.com/datatug/datatug-meta-iso/main/geo/country/country.json),
  [docs](https://github.com/datatug/datatug-meta-iso/tree/main/geo/country)
  - **ID**: string
  - **Name**: string
- **Local** (declared in [Country.entity.json](Country.entity.json), used by this demo project):
  - **Name**: string — `namePatterns: ["Country"]`, declared against the `Country` column
    on Chinook Customer rows and inGitDB support notes.
  - **Currency**: string — the ISO currency code for the country (e.g. `CAD`), used as the
    `Country.Currency` parameter tag on the `currency-rate` HTTP query
    (see [queries/reference](../../queries/reference)).
