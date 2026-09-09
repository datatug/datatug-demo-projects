// Package tests exercises datatug-demo-projects/demo-project-1 against a
// real, tagged datatug-core: it proves the project loads and validates, that
// the declared field mappings plan task 4 (see ~/briefs/s3b-demo-mappings.md)
// listed are actually present, and that datatug-core's pkg/semantic resolver
// and applicability logic (PRs #303/#304) produce the results the
// core-investigation-loop feature's journeys describe for this project.
package tests

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/datatug/datatug-core/pkg/datatug"
	"github.com/datatug/datatug-core/pkg/semantic"
	"github.com/datatug/datatug-core/pkg/storage/filestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const projectDir = "../demo-project-1"

// entityIDs lists demo-project-1's entities (entities/README.md is stale and
// omits several of these; entities/entities-summary.json is a notes-only
// index with no field data) - kept here so TestDemoProject1_Validate
// exercises the real, complete set.
var entityIDs = []string{"Album", "Artist", "Country", "Customer", "Invoice", "InvoiceLine", "Person", "Track"}

func newStore(t *testing.T) datatug.ProjectStore {
	t.Helper()
	return filestore.NewProjectStore("demo-project-1", projectDir)
}

// loadEntity reads one entity file directly from its known on-disk path,
// bypassing filestore.fsEntitiesStore.LoadEntity/LoadEntities.
//
// As of the datatug-core version this module pins (v0.19.0),
// pkg/storage/filestore/store_entities.go still assumes a flat
// "<entities>/<id>.entity.json" layout, but datatug-demo-projects nests each
// entity as "<entities>/<id>/<id>.entity.json" - so LoadEntities finds zero
// entities here, and LoadEntity(id) 404s. datatug-cli's *vendored* copy of
// this same package (pkg/datatug-core/storage/filestore/store_entities.go in
// that repo) already moved to the nested, per-entity-directory convention
// (see its entityDirPath/entityFilePath helpers) and datatug-cli's own
// `entity list`/`entity show` commands work against this exact project as a
// result - but that fix was never carried back to the published
// datatug-core module. This is a discovered datatug-core defect, out of this
// stream's scope to fix (see the PR body); this helper reads the same real
// datatug.Entity type and on-disk file the fixed loader would, just without
// going through the currently-broken directory walk.
func loadEntity(t *testing.T, id string) *datatug.Entity {
	t.Helper()
	filePath := filepath.Join(projectDir, "entities", id, id+".entity.json")
	data, err := os.ReadFile(filePath)
	require.NoError(t, err, "failed to read %s", filePath)
	entity := &datatug.Entity{}
	require.NoError(t, json.Unmarshal(data, entity), "failed to parse %s", filePath)
	return entity
}

func loadAllEntities(t *testing.T) datatug.Entities {
	t.Helper()
	entities := make(datatug.Entities, len(entityIDs))
	for i, id := range entityIDs {
		entities[i] = loadEntity(t, id)
	}
	return entities
}

func fieldByID(t *testing.T, entity *datatug.Entity, fieldID string) *datatug.EntityField {
	t.Helper()
	for _, f := range entity.Fields {
		if f.ID == fieldID {
			return f
		}
	}
	t.Fatalf("field %s.%s not found", entity.ID, fieldID)
	return nil
}

// loadQueries loads the three library queries these tests need, by their
// folder-qualified id (this path works correctly against the published
// datatug-core module - the queries store's LoadQuery takes an explicit
// folder path as part of id, unlike the entities store).
func loadQueries(t *testing.T, store datatug.ProjectStore, ids ...string) datatug.QueryDefs {
	t.Helper()
	ctx := context.Background()
	queries := make(datatug.QueryDefs, len(ids))
	for i, id := range ids {
		q, err := store.LoadQuery(ctx, id)
		require.NoError(t, err, "failed to load query %s", id)
		queries[i] = q
	}
	return queries
}

// TestDemoProject1_Validate assembles the full project (every entity, read
// directly per loadEntity's doc comment, plus its three DTQL/SQL library
// queries) and asserts Project.Validate() passes end to end against real
// datatug-core.
func TestDemoProject1_Validate(t *testing.T) {
	store := newStore(t)
	project, err := store.LoadProject(context.Background())
	require.NoError(t, err, "failed to load %s", projectDir)

	project.Entities = loadAllEntities(t)
	project.Queries = &datatug.QueriesFolder{
		Items: loadQueries(t, store, "customers/customer-invoices", "customers/customer-purchases-by-genre", "invoices/invoice-lines"),
	}

	assert.NoError(t, project.Validate())
}

// TestDemoProject1_DeclaredMappings asserts the five mappings plan task 4
// lists are present, exactly as declared (see s3b-demo-mappings.md item 1).
func TestDemoProject1_DeclaredMappings(t *testing.T) {
	customer := loadEntity(t, "Customer")
	invoice := loadEntity(t, "Invoice")
	country := loadEntity(t, "Country")

	assert.Contains(t, fieldByID(t, customer, "ID").Mappings,
		datatug.PhysicalRef{Source: "chinook", Collection: "Customer", Column: "CustomerId"})
	assert.Contains(t, fieldByID(t, customer, "ID").Mappings,
		datatug.PhysicalRef{Source: "support-notes", Collection: "support-notes", Column: "CustomerId"})
	assert.Contains(t, fieldByID(t, customer, "Email").Mappings,
		datatug.PhysicalRef{Source: "chinook", Collection: "Customer", Column: "Email"})
	assert.Contains(t, fieldByID(t, invoice, "ID").Mappings,
		datatug.PhysicalRef{Source: "chinook", Collection: "Invoice", Column: "InvoiceId"})
	assert.Contains(t, fieldByID(t, country, "Name").Mappings,
		datatug.PhysicalRef{Source: "chinook", Collection: "Customer", Column: "Country"})
}

// TestDemoProject1_ResolveChinookCustomerColumns runs semantic.Resolve over a
// hard-coded Chinook Customer column list, per s3b-demo-mappings.md item 3.
func TestDemoProject1_ResolveChinookCustomerColumns(t *testing.T) {
	entities := []*datatug.Entity{loadEntity(t, "Customer"), loadEntity(t, "Invoice"), loadEntity(t, "Country")}

	columns := []semantic.Column{
		{Name: "CustomerId", Type: "integer"},
		{Name: "Country", Type: "string"},
		{Name: "Email", Type: "string"},
		{Name: "FirstName", Type: "string"}, // no mapping, no pattern -> absent
	}
	got := semantic.Resolve(entities, "chinook", "Customer", columns)

	byColumn := make(map[string]semantic.Resolution, len(got))
	for _, r := range got {
		byColumn[r.Column] = r
	}

	assertDeclared := func(column, entity, field string) {
		r, ok := byColumn[column]
		require.True(t, ok, "expected a resolution for column %q", column)
		assert.Equal(t, entity, r.Entity, "column %q entity", column)
		assert.Equal(t, field, r.Field, "column %q field", column)
		assert.Equal(t, semantic.Declared, r.Provenance, "column %q provenance", column)
	}
	assertDeclared("CustomerId", "Customer", "ID")
	assertDeclared("Country", "Country", "Name")
	assertDeclared("Email", "Customer", "Email")

	_, hasFirstName := byColumn["FirstName"]
	assert.False(t, hasFirstName, "FirstName has no mapping or name pattern and must not resolve")
}

// TestDemoProject1_ApplicableQueries runs semantic.Applicable with
// Customer.ID=5, per s3b-demo-mappings.md item 3.
func TestDemoProject1_ApplicableQueries(t *testing.T) {
	store := newStore(t)
	queries := loadQueries(t, store, "customers/customer-invoices", "customers/customer-purchases-by-genre", "invoices/invoice-lines")
	customerInvoices, customerPurchasesByGenre, invoiceLines := queries[0], queries[1], queries[2]

	assert.Equal(t, datatug.QueryTypeDTQL, customerInvoices.Type, "customer-invoices must have been converted to DTQL")

	available := []semantic.SemanticValue{
		{Entity: "Customer", Field: "ID", Value: 5, Source: "chinook", Collection: "Customer", Column: "CustomerId", Provenance: semantic.Declared},
	}

	applicable, notYet := semantic.Applicable(
		[]*datatug.QueryDef{customerInvoices, customerPurchasesByGenre, invoiceLines},
		available,
	)

	applicableIDs := make([]string, len(applicable))
	for i, a := range applicable {
		applicableIDs[i] = a.Query.ID
	}
	assert.ElementsMatch(t, []string{"customer-invoices", "customer-purchases-by-genre"}, applicableIDs)

	require.Len(t, notYet, 1)
	assert.Equal(t, "invoice-lines", notYet[0].Query.ID)
	assert.Equal(t, []string{"Invoice.ID"}, notYet[0].Missing)
}
