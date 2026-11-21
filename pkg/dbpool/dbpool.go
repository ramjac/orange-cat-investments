package dbpool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultMaxConnLifetime       = 30 * time.Minute
	defaultMaxConnLifetimeJitter = 5 * time.Minute
)

// allowedDomainSchemas enumerates the isolated PostgreSQL schemas owned by OCI domain APIs.
var allowedDomainSchemas = []string{
	"workforce",
	"facilities",
	"core_invest",
	"ops",
}

// ResolveDSN resolves a PostgreSQL DSN, preferring credFilePath when readable and non-empty,
// and falling back to envURL otherwise. Returns an error if neither yields a non-empty DSN.
func ResolveDSN(envURL, credFilePath string) (string, error) {
	var fileReadErr error
	if credFilePath != "" {
		data, err := os.ReadFile(credFilePath)
		if err == nil {
			if trimmed := strings.TrimSpace(string(data)); trimmed != "" {
				return trimmed, nil
			}
		} else {
			fileReadErr = err
		}
	}

	if trimmed := strings.TrimSpace(envURL); trimmed != "" {
		return trimmed, nil
	}

	if fileReadErr != nil {
		return "", fmt.Errorf("dbpool: database DSN unavailable from credentials file or environment: %w", fileReadErr)
	}
	return "", errors.New("dbpool: database DSN is empty in both credentials file and environment")
}

// ConfigureDynamicCredentials configures pool connection lifetime limits and installs a
// BeforeConnect hook that refreshes connection credentials on every new physical connection.
func ConfigureDynamicCredentials(cfg *pgxpool.Config, reloadDSN func() (string, error)) {
	if cfg == nil {
		return
	}
	cfg.MaxConnLifetime = defaultMaxConnLifetime
	cfg.MaxConnLifetimeJitter = defaultMaxConnLifetimeJitter

	if reloadDSN != nil {
		cfg.BeforeConnect = func(_ context.Context, connCfg *pgx.ConnConfig) error {
			latestDSN, err := reloadDSN()
			if err != nil {
				return fmt.Errorf("dbpool: reload DSN: %w", err)
			}
			parsed, err := pgx.ParseConfig(latestDSN)
			if err != nil {
				return fmt.Errorf("dbpool: parse reloaded DSN: %w", err)
			}
			connCfg.User = parsed.User
			connCfg.Password = parsed.Password
			connCfg.Host = parsed.Host
			connCfg.Port = parsed.Port
			connCfg.Database = parsed.Database
			return nil
		}
	}
}

// NewDynamicPool creates a pgxpool.Pool configured with either dynamic credential reloading
// from credFilePath (with envURL fallback) or a static 30-minute connection lifetime.
func NewDynamicPool(ctx context.Context, envURL, credFilePath string) (*pgxpool.Pool, error) {
	initialDSN, err := ResolveDSN(envURL, credFilePath)
	if err != nil {
		return nil, err
	}

	cfg, err := pgxpool.ParseConfig(initialDSN)
	if err != nil {
		return nil, fmt.Errorf("dbpool: parse initial DSN: %w", err)
	}

	if credFilePath != "" {
		ConfigureDynamicCredentials(cfg, func() (string, error) {
			return ResolveDSN(envURL, credFilePath)
		})
	} else {
		cfg.MaxConnLifetime = defaultMaxConnLifetime
	}

	return pgxpool.NewWithConfig(ctx, cfg)
}

// BuildRoleCreationSQL generates OpenBao PostgreSQL creation_statements scoped strictly
// to a single OCI domain schema while revoking privileges on all other domain schemas.
func BuildRoleCreationSQL(schema string) ([]string, error) {
	if !slices.Contains(allowedDomainSchemas, schema) {
		return nil, fmt.Errorf("dbpool: invalid domain schema %q", schema)
	}

	stmts := make([]string, 0, 9)
	stmts = append(stmts, `CREATE ROLE "{{name}}" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}';`)

	for _, other := range allowedDomainSchemas {
		if other == schema {
			continue
		}
		stmts = append(stmts, fmt.Sprintf(`REVOKE ALL PRIVILEGES ON SCHEMA %s FROM "{{name}}";`, other))
	}

	stmts = append(stmts,
		fmt.Sprintf(`GRANT USAGE ON SCHEMA %s TO "{{name}}";`, schema),
		fmt.Sprintf(`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %s TO "{{name}}";`, schema),
		fmt.Sprintf(`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA %s TO "{{name}}";`, schema),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO "{{name}}";`, schema),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT USAGE, SELECT ON SEQUENCES TO "{{name}}";`, schema),
	)

	return stmts, nil
}
