//go:build integration

package repository

import (
	"context"
	"database/sql"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestSupplierMCPMigrationsFreshAndUpgrade(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, selectDockerImage(ctx, postgresImageTag),
		tcpostgres.WithDatabase("supplier_upgrade"), tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		require.NoError(t, container.Terminate(cleanupCtx))
	})
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	// Reconstruct the previously shipped migration set without renaming either 249.
	previous := fstest.MapFS{}
	entries, err := migrations.FS.ReadDir(".")
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "249_mcp_oauth.sql" {
			continue
		}
		data, readErr := migrations.FS.ReadFile(entry.Name())
		require.NoError(t, readErr)
		previous[entry.Name()] = &fstest.MapFile{Data: data}
	}
	require.NoError(t, applyMigrationsFS(ctx, db, previous))
	var supplierID, accountID int64
	require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO suppliers (name, base_url, balance)
		VALUES ('upgrade fixture', 'https://supplier.example', 123.45) RETURNING id`).Scan(&supplierID))
	require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO accounts (name, platform, type, supplier_id, supplier_disabled)
		VALUES ('existing key', 'openai', 'apikey', $1, TRUE) RETURNING id`, supplierID).Scan(&accountID))
	var oldChecksum string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations
		WHERE filename = '249_supplier_management.sql'`).Scan(&oldChecksum))

	require.NoError(t, ApplyMigrations(ctx, db))
	require.NoError(t, ApplyMigrations(ctx, db), "a second startup must not reapply either migration")
	var checksum, balance string
	var disabled bool
	require.NoError(t, db.QueryRowContext(ctx, `SELECT s.balance::text, a.supplier_disabled
		FROM suppliers s JOIN accounts a ON a.supplier_id = s.id WHERE a.id = $1`, accountID).Scan(&balance, &disabled))
	require.Equal(t, "123.45000000", balance)
	require.True(t, disabled)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations
		WHERE filename = '249_supplier_management.sql'`).Scan(&checksum))
	require.Equal(t, oldChecksum, checksum)

	// The shared integration database was initialized from the complete merged set.
	for _, database := range []*sql.DB{db, integrationDB} {
		var count int
		require.NoError(t, database.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations
			WHERE filename IN ('249_supplier_management.sql', '249_mcp_oauth.sql')`).Scan(&count))
		require.Equal(t, 2, count)
		for _, table := range []string{"suppliers", "mcp_oauth_grants", "mcp_oauth_codes", "mcp_oauth_credentials"} {
			var relation sql.NullString
			require.NoError(t, database.QueryRowContext(ctx, "SELECT to_regclass($1)::text", "public."+table).Scan(&relation))
			require.True(t, relation.Valid, "expected %s after migration", table)
		}
	}
}
