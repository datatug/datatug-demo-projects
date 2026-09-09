package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/dal-go/dalgo/access"
	"github.com/dal-go/dalgo/dal"
	"github.com/datatug/datatug-cli/pkg/accesspolicies"
	"github.com/stretchr/testify/require"
)

// stubReadSession is a minimal dal.ReadSession whose only implemented method
// returns no rows: this test is about the policy DECISION - may a given
// principal even reach the native-SQL resource - not about executing real
// SQL against a real database. access.SecureReadSession only calls through
// to the wrapped session once a request is allowed, so a denial never
// touches this stub at all (mirrors datatug-cli's own
// pkg/accesspolicies.stubSession test helper).
type stubReadSession struct {
	dal.ReadSession
}

func (stubReadSession) ExecuteQueryToRecordsReader(_ context.Context, _ dal.Query) (dal.RecordsReader, error) {
	return nil, nil
}

// TestDemoProject1_OpaqueQueryPolicy proves demo-project-1/policies/
// customers.yaml grants admin - but not support - the native-SQL resource
// kind (access.OpaqueQuery / opaqueQuery: true), a separate resource kind
// from the structured path:/** scope admin already has (see datatug-cli PR
// #206 body item 3: as originally authored, the policy denied the demo SQL
// queries, e.g. customer-purchases-by-genre, to every principal, admin
// included). Loads the real policy set through the same loader
// (accesspolicies.LoadDir) and dispatch (accesspolicies.Run) datatug-cli
// itself uses, against the real SQL text of demo-project-1's
// customer-purchases-by-genre query.
func TestDemoProject1_OpaqueQueryPolicy(t *testing.T) {
	ctx := context.Background()

	store := newStore(t)
	query, err := store.LoadQuery(ctx, "customers/customer-purchases-by-genre")
	require.NoError(t, err, "failed to load customer-purchases-by-genre")
	require.NotEmpty(t, query.Text)

	loaded, err := accesspolicies.LoadDir("../demo-project-1/policies")
	require.NoError(t, err)

	sqlQuery := dal.NewTextQuery(query.Text, nil)
	session := stubReadSession{}

	t.Run("admin_may_run", func(t *testing.T) {
		_, err := accesspolicies.Run(ctx, session, sqlQuery, accesspolicies.Options{
			Principal: &access.Principal{Roles: []string{"admin"}},
			Policies:  loaded,
		})
		require.NoError(t, err, "admin must be allowed to run the demo SQL query")
	})

	t.Run("support_may_not_run", func(t *testing.T) {
		_, err := accesspolicies.Run(ctx, session, sqlQuery, accesspolicies.Options{
			Principal: &access.Principal{Roles: []string{"support"}},
			Policies:  loaded,
		})
		require.Error(t, err)
		require.True(t, errors.Is(err, access.ErrAccessDenied), "support must be denied the demo SQL query, got: %v", err)
	})
}
