// Package tests exercises datatug-demo-projects/demo-project-1 against a
// real, tagged datatug-core: it proves the project loads and validates, that
// the declared field mappings plan task 4 (see ~/briefs/s3b-demo-mappings.md)
// listed are actually present, and that datatug-core's pkg/semantic resolver
// and applicability logic (PRs #303/#304) produce the results the
// core-investigation-loop feature's journeys describe for this project.
package tests

import (
	"context"
	"testing"

	"github.com/datatug/datatug-core/pkg/datatug"
	"github.com/datatug/datatug-core/pkg/semantic"
	"github.com/datatug/datatug-core/pkg/storage/filestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const projectDir = "../demo-project-1"

func newStore(t *testing.T) datatug.ProjectStore {
	t.Helper()
	return filestore.NewProjectStore("demo-project-1", projectDir)
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
// folder-qualified id. datatug-core's Project.LoadProject does not populate
// Project.Queries at all (a separate, still-open gap, unrelated to the
// entities/boards/dbmodels dual-layout fixes this module now pins) - so
// tests that need query definitions load them directly.
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

// TestDemoProject1_Validate loads demo-project-1 through the real
// filestore.LoadProject (datatug-core v0.20.0/#305 fixed entities loading
// from demo-project-1's per-entity directories; v0.21.0/#306 fixed the same
// bug class for boards and DB models, v0.22.0 makes LoadProject load each
// environment's own "<id>.env.json" and fixes #307 so a file-based sqlite3
// ServerRef with an empty host validates, and Project.Validate() has
// covered declared mappings since v0.17.0/#302) and asserts: every entity
// loads, the demo's board1 board and chinook DB model load with their real
// content, every environment loads with a validating sqlite3 ServerRef, and
// - once its three library queries are attached, the one thing LoadProject
// still doesn't populate - Project.Validate() passes.
func TestDemoProject1_Validate(t *testing.T) {
	store := newStore(t)
	project, err := store.LoadProject(context.Background())
	require.NoError(t, err, "failed to load %s", projectDir)

	wantEntityIDs := []string{"Album", "Artist", "Country", "Customer", "Invoice", "InvoiceLine", "Person", "Track"}
	assert.ElementsMatch(t, wantEntityIDs, project.Entities.IDs())

	if assert.Len(t, project.Boards, 1) {
		assert.Equal(t, "board1", project.Boards[0].ID)
		assert.Equal(t, "1st board", project.Boards[0].Title)
	}

	if assert.Len(t, project.DbModels, 1) {
		assert.Equal(t, "chinook", project.DbModels[0].ID)
	}

	wantEnvIDs := []string{"dev", "local", "prod", "QA", "UAT"}
	assert.ElementsMatch(t, wantEnvIDs, project.Environments.IDs())

	project.Queries = &datatug.QueriesFolder{
		Items: loadQueries(t, store, "customers/customer-invoices", "customers/customer-purchases-by-genre", "invoices/invoice-lines"),
	}

	assert.NoError(t, project.Validate())
}

// TestDemoProject1_Environments loads every demo-project-1 environment
// individually and asserts its dbServers.sqlite3 ServerRef is valid (S42's
// fix for #307: environments/*/*.env.json declared "host":"localhost" for a
// sqlite3 server, which ServerRef.Validate() correctly rejects - sqlite3 is
// file-based and must have an empty host), then resolves the "local"
// environment's "chinook-local" database catalog by id.
func TestDemoProject1_Environments(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	envs, err := store.LoadEnvironments(ctx)
	require.NoError(t, err)

	wantEnvIDs := []string{"dev", "local", "prod", "QA", "UAT"}
	assert.ElementsMatch(t, wantEnvIDs, envs.IDs())

	for _, env := range envs {
		t.Run(env.ID, func(t *testing.T) {
			require.NotEmpty(t, env.DbServers, "environment %s has no dbServers", env.ID)
			for i, server := range env.DbServers {
				assert.NoError(t, server.ServerRef.Validate(), "dbServers[%d].ServerRef in environment %s", i, env.ID)
			}
			assert.NoError(t, env.Validate(), "environment %s", env.ID)
		})
	}

	catalogs, err := store.LoadEnvDbCatalogs(ctx, "local")
	require.NoError(t, err)
	assert.Contains(t, catalogs.IDs(), "chinook-local",
		"the local environment must resolve its chinook-local database catalog")
}

// TestDemoProject1_DeclaredMappings asserts the five mappings plan task 4
// lists are present, exactly as declared (see s3b-demo-mappings.md item 1).
func TestDemoProject1_DeclaredMappings(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	loadEntity := func(id string) *datatug.Entity {
		e, err := store.LoadEntity(ctx, id)
		require.NoError(t, err, "failed to load entity %s", id)
		return e
	}
	customer := loadEntity("Customer")
	invoice := loadEntity("Invoice")
	country := loadEntity("Country")

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
	store := newStore(t)
	entities, err := store.LoadEntities(context.Background())
	require.NoError(t, err)

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
