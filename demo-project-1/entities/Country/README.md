# Entity: Country

## Fields

- **Exported**:
  [JSON](https://raw.githubusercontent.com/datatug/datatug-meta-iso/main/geo/country/country.json),
  [docs](https://github.com/datatug/datatug-meta-iso/tree/main/geo/country)
  - **ID**: string
  - **Name**: string
- **Local** (declared in [Country.entity.json](Country.entity.json), used by this demo project):
  - **Name**: string — `namePatterns: ["Country"]`, matches the `Country` column on `Customer`/`Invoice`
    (chinook) so the semantic resolver can map it once field mappings land.
  - **Currency**: string — the ISO currency code for the country (e.g. `CAD`), used as the
    `Country.Currency` parameter tag on the `currency-rate` HTTP query
    (see [queries/reference](../../queries/reference)).
