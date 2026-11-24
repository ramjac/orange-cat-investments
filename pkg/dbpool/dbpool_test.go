package dbpool

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveDSN(t *testing.T) {
	t.Run("credentials file overrides env URL and trims whitespace", func(t *testing.T) {
		dir := t.TempDir()
		credFile := filepath.Join(dir, "db-url")
		err := os.WriteFile(credFile, []byte("  postgres://dynuser:dynpass@db.internal:5432/oci?sslmode=require \n"), 0o600)
		require.NoError(t, err)

		dsn, err := ResolveDSN("postgres://envuser:envpass@localhost:5432/oci", credFile)
		require.NoError(t, err)
		assert.Equal(t, "postgres://dynuser:dynpass@db.internal:5432/oci?sslmode=require", dsn)
	})

	t.Run("falls back to env URL when credentials file is missing", func(t *testing.T) {
		missingFile := filepath.Join(t.TempDir(), "does-not-exist")

		dsn, err := ResolveDSN("  postgres://envuser:envpass@localhost:5432/oci  ", missingFile)
		require.NoError(t, err)
		assert.Equal(t, "postgres://envuser:envpass@localhost:5432/oci", dsn)
	})

	t.Run("falls back to env URL when credentials file is empty or whitespace", func(t *testing.T) {
		dir := t.TempDir()
		credFile := filepath.Join(dir, "empty-url")
		require.NoError(t, os.WriteFile(credFile, []byte("   \n\t "), 0o600))

		dsn, err := ResolveDSN("postgres://envuser:envpass@localhost:5432/oci", credFile)
		require.NoError(t, err)
		assert.Equal(t, "postgres://envuser:envpass@localhost:5432/oci", dsn)
	})

	t.Run("uses env URL when credentials file path is empty", func(t *testing.T) {
		dsn, err := ResolveDSN("postgres://envuser:envpass@localhost:5432/oci", "")
		require.NoError(t, err)
		assert.Equal(t, "postgres://envuser:envpass@localhost:5432/oci", dsn)
	})

	t.Run("returns error when both credentials file and env URL are empty", func(t *testing.T) {
		_, err := ResolveDSN("", "")
		require.Error(t, err)

		emptyFile := filepath.Join(t.TempDir(), "empty")
		require.NoError(t, os.WriteFile(emptyFile, []byte(""), 0o600))
		_, err = ResolveDSN("   ", emptyFile)
		require.Error(t, err)

		missingFile := filepath.Join(t.TempDir(), "missing")
		_, err = ResolveDSN("", missingFile)
		require.Error(t, err)
	})
}

func TestConfigureDynamicCredentials(t *testing.T) {
	t.Run("updates connection config dynamically when credentials file rotates", func(t *testing.T) {
		dir := t.TempDir()
		credFile := filepath.Join(dir, "dsn")
		initialDSN := "postgres://role_v1:secret_v1@pg-primary:5432/oci_db"
		require.NoError(t, os.WriteFile(credFile, []byte(initialDSN), 0o600))

		cfg, err := pgxpool.ParseConfig(initialDSN)
		require.NoError(t, err)

		ConfigureDynamicCredentials(cfg, func() (string, error) {
			return ResolveDSN("", credFile)
		})

		assert.Equal(t, 30*time.Minute, cfg.MaxConnLifetime)
		assert.Equal(t, 5*time.Minute, cfg.MaxConnLifetimeJitter)
		require.NotNil(t, cfg.BeforeConnect)

		connCfg := cfg.ConnConfig.Copy()
		err = cfg.BeforeConnect(context.Background(), connCfg)
		require.NoError(t, err)
		assert.Equal(t, "role_v1", connCfg.User)
		assert.Equal(t, "secret_v1", connCfg.Password)
		assert.Equal(t, "pg-primary", connCfg.Host)
		assert.Equal(t, uint16(5432), connCfg.Port)
		assert.Equal(t, "oci_db", connCfg.Database)

		// Simulate OpenBao rotating the leased role and password in the file.
		rotatedDSN := "postgres://role_v2:secret_v2@pg-standby:5433/oci_db_v2"
		require.NoError(t, os.WriteFile(credFile, []byte(rotatedDSN+"\n"), 0o600))

		err = cfg.BeforeConnect(context.Background(), connCfg)
		require.NoError(t, err)
		assert.Equal(t, "role_v2", connCfg.User)
		assert.Equal(t, "secret_v2", connCfg.Password)
		assert.Equal(t, "pg-standby", connCfg.Host)
		assert.Equal(t, uint16(5433), connCfg.Port)
		assert.Equal(t, "oci_db_v2", connCfg.Database)
	})

	t.Run("sets lifetimes without BeforeConnect when reloadDSN is nil", func(t *testing.T) {
		cfg, err := pgxpool.ParseConfig("postgres://user:pass@localhost:5432/oci")
		require.NoError(t, err)

		ConfigureDynamicCredentials(cfg, nil)
		assert.Equal(t, 30*time.Minute, cfg.MaxConnLifetime)
		assert.Equal(t, 5*time.Minute, cfg.MaxConnLifetimeJitter)
		assert.Nil(t, cfg.BeforeConnect)
	})

	t.Run("returns error when reloadDSN fails or returns invalid DSN", func(t *testing.T) {
		cfg, err := pgxpool.ParseConfig("postgres://user:pass@localhost:5432/oci")
		require.NoError(t, err)

		ConfigureDynamicCredentials(cfg, func() (string, error) {
			return "", errors.New("reload boom")
		})
		err = cfg.BeforeConnect(context.Background(), cfg.ConnConfig.Copy())
		require.Error(t, err)

		ConfigureDynamicCredentials(cfg, func() (string, error) {
			return "://bad-dsn", nil
		})
		err = cfg.BeforeConnect(context.Background(), cfg.ConnConfig.Copy())
		require.Error(t, err)
	})
}

func TestNewDynamicPool(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error when DSN is missing", func(t *testing.T) {
		pool, err := NewDynamicPool(ctx, "", "")
		require.Error(t, err)
		assert.Nil(t, pool)
	})

	t.Run("configures BeforeConnect when credFilePath is provided", func(t *testing.T) {
		credFile := filepath.Join(t.TempDir(), "dsn")
		require.NoError(t, os.WriteFile(credFile, []byte("postgres://u1:p1@localhost:5432/oci"), 0o600))

		pool, err := NewDynamicPool(ctx, "", credFile)
		require.NoError(t, err)
		defer pool.Close()

		assert.Equal(t, 30*time.Minute, pool.Config().MaxConnLifetime)
		assert.Equal(t, 5*time.Minute, pool.Config().MaxConnLifetimeJitter)
		assert.NotNil(t, pool.Config().BeforeConnect)
	})

	t.Run("configures static MaxConnLifetime when credFilePath is empty", func(t *testing.T) {
		pool, err := NewDynamicPool(ctx, "postgres://u1:p1@localhost:5432/oci", "")
		require.NoError(t, err)
		defer pool.Close()

		assert.Equal(t, 30*time.Minute, pool.Config().MaxConnLifetime)
		assert.Nil(t, pool.Config().BeforeConnect)
	})
}

func TestBuildRoleCreationSQL(t *testing.T) {
	validSchemas := []string{"workforce", "facilities", "core_invest", "ops"}

	for _, target := range validSchemas {
		t.Run("valid schema "+target, func(t *testing.T) {
			stmts, err := BuildRoleCreationSQL(target)
			require.NoError(t, err)
			require.NotEmpty(t, stmts)

			joined := strings.Join(stmts, "\n")
			assert.Contains(t, joined, `CREATE ROLE "{{name}}" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}';`)
			assert.Contains(t, joined, "GRANT USAGE ON SCHEMA "+target+` TO "{{name}}";`)
			assert.Contains(t, joined, "GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA "+target+` TO "{{name}}";`)
			assert.Contains(t, joined, "GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA "+target+` TO "{{name}}";`)
			assert.Contains(t, joined, "ALTER DEFAULT PRIVILEGES IN SCHEMA "+target+` GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO "{{name}}";`)
			assert.Contains(t, joined, "ALTER DEFAULT PRIVILEGES IN SCHEMA "+target+` GRANT USAGE, SELECT ON SEQUENCES TO "{{name}}";`)

			for _, other := range validSchemas {
				if other == target {
					assert.NotContains(t, joined, "REVOKE ALL PRIVILEGES ON SCHEMA "+target)
				} else {
					assert.Contains(t, joined, "REVOKE ALL PRIVILEGES ON SCHEMA "+other+` FROM "{{name}}";`)
					assert.NotContains(t, joined, "GRANT USAGE ON SCHEMA "+other)
				}
			}
		})
	}

	invalidInputs := []string{
		"public",
		"unknown",
		"",
		"workforce; DROP TABLE users; --",
		"WORKFORCE",
		"core-invest",
	}
	for _, invalid := range invalidInputs {
		t.Run("invalid schema "+invalid, func(t *testing.T) {
			stmts, err := BuildRoleCreationSQL(invalid)
			require.Error(t, err)
			assert.Nil(t, stmts)
		})
	}
}
