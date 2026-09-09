package tests

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dal-go/dalgo/access"
	"github.com/dal-go/dalgo/dal"
	"github.com/datatug/datatug-cli/pkg/accesspolicies"
	"github.com/datatug/datatug-cli/pkg/secureread"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type acceptanceCustomer struct {
	ID               int    `json:"id"`
	Country          string `json:"country"`
	InvoiceCount     int    `json:"invoiceCount"`
	SupportNoteCount int    `json:"supportNoteCount"`
}

type acceptanceFixture struct {
	Database struct {
		Repository string `json:"repository"`
		Revision   string `json:"revision"`
		Path       string `json:"path"`
		SHA256     string `json:"sha256"`
	} `json:"database"`
	Customers struct {
		Brazilian     acceptanceCustomer `json:"brazilian"`
		Canadian      acceptanceCustomer `json:"canadian"`
		Investigation acceptanceCustomer `json:"investigation"`
	} `json:"customers"`
	SupportScope struct {
		Country          string `json:"country"`
		CustomerCount    int    `json:"customerCount"`
		InvoiceCount     int    `json:"invoiceCount"`
		SupportNoteCount int    `json:"supportNoteCount"`
	} `json:"supportScope"`
}

func loadAcceptanceFixture(t *testing.T) acceptanceFixture {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(projectDir, "fixtures", "chinook", "phase1-acceptance.json"))
	require.NoError(t, err)
	var fixture acceptanceFixture
	require.NoError(t, json.Unmarshal(b, &fixture))
	return fixture
}

type supportNoteFact struct {
	CustomerID int
	Country    string
}

func integerValue(t *testing.T, value any, field string) int {
	t.Helper()
	switch value := value.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		converted := int(value)
		require.Equal(t, value, float64(converted), "%s must be an integer", field)
		return converted
	default:
		t.Fatalf("%s has type %T", field, value)
		return 0
	}
}

func loadSupportNoteFacts(t *testing.T) []supportNoteFact {
	t.Helper()
	root, err := filepath.Abs(filepath.Join(projectDir, "data", "ingitdb"))
	require.NoError(t, err)
	executor := secureread.NewExecutor(secureread.Session{Unrestricted: true})
	query := dal.NewQueryBuilder(dal.From(dal.NewRootCollectionRef("support-notes", ""))).
		SelectColumns(dal.Column{Expression: dal.Field("CustomerId")}, dal.Column{Expression: dal.Field("Country")})
	result, err := executor.RunStructured(context.Background(), "ingitdb://"+root, query, nil)
	require.NoError(t, err)
	require.Len(t, result.Rows, 8)

	facts := make([]supportNoteFact, 0, len(result.Rows))
	for _, row := range result.Rows {
		customerID := integerValue(t, row.Data["CustomerId"], "CustomerId")
		country, ok := row.Data["Country"].(string)
		require.True(t, ok, "Country has type %T", row.Data["Country"])
		facts = append(facts, supportNoteFact{CustomerID: customerID, Country: country})
	}
	return facts
}

func TestPhase1AcceptanceFixtureConsistency(t *testing.T) {
	fixture := loadAcceptanceFixture(t)

	require.NotEmpty(t, fixture.Database.Repository)
	require.Len(t, fixture.Database.Revision, 40)
	require.NotEmpty(t, fixture.Database.Path)
	require.Len(t, fixture.Database.SHA256, 64)
	require.Equal(t, fixture.Customers.Canadian.Country, fixture.SupportScope.Country)
	require.NotEqual(t, fixture.Customers.Brazilian.ID, fixture.Customers.Canadian.ID)
	require.NotEqual(t, fixture.Customers.Canadian.ID, fixture.Customers.Investigation.ID)

	notesByCustomer := map[int]int{}
	notesByCountry := map[string]int{}
	for _, note := range loadSupportNoteFacts(t) {
		notesByCustomer[note.CustomerID]++
		notesByCountry[note.Country]++
	}
	assert.Equal(t, fixture.Customers.Brazilian.SupportNoteCount, notesByCustomer[fixture.Customers.Brazilian.ID])
	assert.Equal(t, fixture.Customers.Canadian.SupportNoteCount, notesByCustomer[fixture.Customers.Canadian.ID])
	assert.Equal(t, fixture.Customers.Investigation.SupportNoteCount, notesByCustomer[fixture.Customers.Investigation.ID])
	assert.Equal(t, fixture.SupportScope.SupportNoteCount, notesByCountry[fixture.SupportScope.Country])
}

// TestPinnedChinookDatabase derives the IDs and counts in the acceptance
// fixture from the immutable SQLite file itself. DATATUG_CHINOOK_DB is an
// explicit acceptance prerequisite; the ordinary demo-project unit suite
// validates only the committed derived fixture when the owning database
// repository is not checked out beside it.
func TestPinnedChinookDatabase(t *testing.T) {
	dbPath := os.Getenv("DATATUG_CHINOOK_DB")
	if dbPath == "" {
		if run := flag.Lookup("test.run"); run != nil && strings.Contains(run.Value.String(), "TestPinnedChinookDatabase") {
			t.Fatal("DATATUG_CHINOOK_DB is required when the pinned database verification is selected explicitly")
		}
		t.Skip("DATATUG_CHINOOK_DB is required for pinned Chinook database verification")
	}
	fixture := loadAcceptanceFixture(t)

	f, err := os.Open(dbPath)
	require.NoError(t, err)
	hash := sha256.New()
	_, err = io.Copy(hash, f)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.Equal(t, fixture.Database.SHA256, fmt.Sprintf("%x", hash.Sum(nil)))

	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()

	for name, customer := range map[string]acceptanceCustomer{
		"brazilian":     fixture.Customers.Brazilian,
		"canadian":      fixture.Customers.Canadian,
		"investigation": fixture.Customers.Investigation,
	} {
		t.Run(name, func(t *testing.T) {
			var country string
			require.NoError(t, db.QueryRow("SELECT Country FROM Customer WHERE CustomerId = ?", customer.ID).Scan(&country))
			assert.Equal(t, customer.Country, country)

			var invoiceCount int
			require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM Invoice WHERE CustomerId = ?", customer.ID).Scan(&invoiceCount))
			assert.Equal(t, customer.InvoiceCount, invoiceCount)
		})
	}

	var customerCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM Customer WHERE Country = ?", fixture.SupportScope.Country).Scan(&customerCount))
	assert.Equal(t, fixture.SupportScope.CustomerCount, customerCount)

	var invoiceCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM Invoice WHERE BillingCountry = ?", fixture.SupportScope.Country).Scan(&invoiceCount))
	assert.Equal(t, fixture.SupportScope.InvoiceCount, invoiceCount)
}

func TestPhase1HTTPFixtureUsesRetrievedCurrency(t *testing.T) {
	fixture := loadAcceptanceFixture(t)

	var facts struct {
		Data struct {
			Name     string `json:"name"`
			Currency string `json:"currency"`
		} `json:"data"`
	}
	b, err := os.ReadFile(filepath.Join(projectDir, "fixtures", "http", "country-facts.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &facts))
	assert.Equal(t, fixture.Customers.Canadian.Country, facts.Data.Name)
	require.NotEmpty(t, facts.Data.Currency)

	var rate struct {
		Rates map[string]float64 `json:"rates"`
	}
	b, err = os.ReadFile(filepath.Join(projectDir, "fixtures", "http", "currency-rate.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &rate))
	assert.Contains(t, rate.Rates, facts.Data.Currency,
		"the rate fixture must use the currency returned by the selected Country lookup")
}

type policyRecordingSession struct {
	dal.ReadSession
	query dal.Query
}

func (s *policyRecordingSession) ExecuteQueryToRecordsReader(_ context.Context, query dal.Query) (dal.RecordsReader, error) {
	s.query = query
	return nil, nil
}

func runSupportQuery(t *testing.T, collection string, customerID int) (accesspolicies.Result, dal.StructuredQuery) {
	t.Helper()
	loaded, err := accesspolicies.LoadDir(filepath.Join(projectDir, "policies"))
	require.NoError(t, err)

	builder := dal.NewQueryBuilder(dal.From(dal.NewRootCollectionRef(collection, "")))
	if customerID != 0 {
		builder.WhereField("CustomerId", dal.Equal, customerID)
	}
	query := builder.SelectColumns()
	session := &policyRecordingSession{}
	result, err := accesspolicies.Run(context.Background(), session, query, accesspolicies.Options{
		Principal: &access.Principal{Roles: []string{"support"}},
		Policies:  loaded,
	})
	require.NoError(t, err)
	require.NotNil(t, session.query)
	rewritten, ok := session.query.(dal.StructuredQuery)
	require.True(t, ok, "rewritten query has type %T", session.query)
	return result, rewritten
}

func selectedFieldNames(t *testing.T, query dal.StructuredQuery) []string {
	t.Helper()
	names := make([]string, 0, len(query.Columns()))
	for _, column := range query.Columns() {
		field, ok := column.Expression.(dal.FieldRef)
		require.True(t, ok, "projected expression has type %T", column.Expression)
		names = append(names, field.Name())
	}
	return names
}

func TestDemoProject1_SupportPoliciesAreIndependent(t *testing.T) {
	t.Run("Customer hides Email and filters Canada", func(t *testing.T) {
		result, query := runSupportQuery(t, "Customer", 0)
		require.Len(t, result.Lines, 1)
		assert.Contains(t, result.Lines[0].String(), `rule "support/customers-support"`)
		assert.Equal(t, "Country = 'Canada'", query.Where().String())
		assert.NotContains(t, selectedFieldNames(t, query), "Email")
	})

	t.Run("Invoice adds its own Canada restriction", func(t *testing.T) {
		result, query := runSupportQuery(t, "Invoice", 1)
		require.Len(t, result.Lines, 1)
		assert.Contains(t, result.Lines[0].String(), `rule "support/invoices-support"`)
		assert.Equal(t, "(CustomerId = 1 AND BillingCountry = 'Canada')", query.Where().String())
	})

	t.Run("support notes add their own Canada restriction", func(t *testing.T) {
		result, query := runSupportQuery(t, "support-notes", 1)
		require.Len(t, result.Lines, 1)
		assert.Contains(t, result.Lines[0].String(), `rule "support/support-notes-support"`)
		assert.Equal(t, "(CustomerId = 1 AND Country = 'Canada')", query.Where().String())
	})
}

func TestDemoProject1_SupportNotesPolicyRunsAgainstRealInGitDB(t *testing.T) {
	session, err := secureread.NewSession(secureread.SessionOptions{
		As:          "support-agent",
		Roles:       []string{"support"},
		PoliciesDir: filepath.Join(projectDir, "policies"),
	})
	require.NoError(t, err)
	executor := secureread.NewExecutor(session)
	root, err := filepath.Abs(filepath.Join(projectDir, "data", "ingitdb"))
	require.NoError(t, err)

	run := func(t *testing.T, customerID int) secureread.Result {
		t.Helper()
		query := dal.NewQueryBuilder(dal.From(dal.NewRootCollectionRef("support-notes", ""))).
			WhereField("CustomerId", dal.Equal, customerID).
			SelectColumns()
		result, err := executor.RunStructured(context.Background(), "ingitdb://"+root, query, nil)
		require.NoError(t, err)
		assert.Contains(t, result.Limitations, secureread.Limitation{Kind: secureread.LimitationRowsFiltered})
		return result
	}

	t.Run("Brazilian note is filtered", func(t *testing.T) {
		assert.Empty(t, run(t, 1).Rows)
	})

	t.Run("Canadian note is returned", func(t *testing.T) {
		result := run(t, 3)
		require.Len(t, result.Rows, 1)
		assert.Equal(t, "Canada", result.Rows[0].Data["Country"])
		assert.EqualValues(t, 3, result.Rows[0].Data["CustomerId"])
	})
}

func TestDemoProject1_SupportCannotProbeEmailOrUnknownSources(t *testing.T) {
	loaded, err := accesspolicies.LoadDir(filepath.Join(projectDir, "policies"))
	require.NoError(t, err)
	principal := &access.Principal{Roles: []string{"support"}}

	t.Run("Email", func(t *testing.T) {
		session := &policyRecordingSession{}
		query := dal.NewQueryBuilder(dal.From(dal.NewRootCollectionRef("Customer", ""))).
			SelectColumns(dal.Column{Expression: dal.Field("CustomerId")}, dal.Column{Expression: dal.Field("Email")})
		_, err := accesspolicies.Run(context.Background(), session, query, accesspolicies.Options{Principal: principal, Policies: loaded})
		require.Error(t, err)
		assert.True(t, errors.Is(err, access.ErrAccessDenied))
		assert.Nil(t, session.query, "a hidden-field probe must not reach the data source")
	})

	t.Run("unknown source", func(t *testing.T) {
		session := &policyRecordingSession{}
		query := dal.NewQueryBuilder(dal.From(dal.NewRootCollectionRef("Employee", ""))).SelectColumns()
		_, err := accesspolicies.Run(context.Background(), session, query, accesspolicies.Options{Principal: principal, Policies: loaded})
		require.Error(t, err)
		assert.True(t, errors.Is(err, access.ErrAccessDenied))
		assert.Nil(t, session.query, "an unauthorized source must not receive a query")
	})
}
