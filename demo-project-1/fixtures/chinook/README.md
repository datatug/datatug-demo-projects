# Pinned Chinook acceptance data

`phase1-acceptance.json` records the immutable Chinook SQLite input used by the Phase 1
demo. The database stays in its owning `datatug/chinook-database` repository; this demo
does not vendor another copy. Before transport acceptance, check out the recorded
revision and verify the database file:

```sh
shasum -a 256 ChinookDatabase/DataSources/Chinook_Sqlite.sqlite
```

The selected rows and counts come from these read-only queries against that file:

```sql
SELECT CustomerId, Country
FROM Customer
WHERE CustomerId IN (1, 3, 5)
ORDER BY CustomerId;

SELECT CustomerId, COUNT(*) AS InvoiceCount
FROM Invoice
WHERE CustomerId IN (1, 3, 5)
GROUP BY CustomerId
ORDER BY CustomerId;

SELECT COUNT(*) AS CustomerCount
FROM Customer
WHERE Country = 'Canada';

SELECT COUNT(*) AS InvoiceCount
FROM Invoice
WHERE BillingCountry = 'Canada';
```

They yield Brazilian customer 1, Canadian customer 3, and investigation customer 5;
each has 7 invoices. Canada has 8 customers and 56 invoices. The committed inGitDB
records independently yield one Canadian support note, one note for customer 1, one for
customer 3, and two for customer 5.

The restricted transport cases therefore use customer 1 to prove an unauthorized
Brazilian lookup returns no rows and customer 3 to prove an authorized Canadian lookup
returns 7 invoices and one support note. The unrestricted J2 lookup keeps customer 5,
whose expected related counts are 7 invoices and 2 support notes.

The normal unit suite checks the committed derived fixture against the real inGitDB and
HTTP fixtures. It does not claim to have opened the external Chinook database. The
database verification is explicit and fails on a hash, country or count mismatch:

```sh
DATATUG_CHINOOK_DB=/path/to/Chinook_Sqlite.sqlite \
  go -C tests test . -run TestPinnedChinookDatabase
```

Selecting this test by name without `DATATUG_CHINOOK_DB` fails, so an acceptance
harness cannot silently turn a missing pinned database into a successful check.
