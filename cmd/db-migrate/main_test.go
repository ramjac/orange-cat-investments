package main

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSQLDB struct {
	execCalls   []string
	applied     map[int64]bool
	appliedSeq  []int64
	stmtsByVer  map[int64][]string
}

func (f *fakeSQLDB) Exec(_ context.Context, sql string, _ ...any) error {
	f.execCalls = append(f.execCalls, sql)
	return nil
}

func (f *fakeSQLDB) QueryApplied(_ context.Context) (map[int64]bool, error) {
	out := make(map[int64]bool, len(f.applied))
	for k, v := range f.applied {
		out[k] = v
	}
	return out, nil
}

func (f *fakeSQLDB) ApplyMigration(_ context.Context, version int64, statements []string) error {
	f.applied[version] = true
	f.appliedSeq = append(f.appliedSeq, version)
	f.stmtsByVer[version] = append([]string(nil), statements...)
	return nil
}

func TestParseGooseUpStatements(t *testing.T) {
	t.Run("plain and StatementBegin dollar-quoted blocks", func(t *testing.T) {
		sql := `
-- header comment
-- +goose Up
ALTER TABLE ops.test ADD COLUMN IF NOT EXISTS note TEXT;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'x') THEN
        CREATE ROLE x;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION ops.fn_test()
RETURNS TRIGGER AS $$
BEGIN
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION ops.fn_test();
`
		stmts := parseGooseUpStatements(sql)
		require.Len(t, stmts, 3)
		assert.Contains(t, stmts[0], "ALTER TABLE ops.test")
		assert.Contains(t, stmts[1], "DO $$")
		assert.Contains(t, stmts[1], "END $$;")
		assert.Contains(t, stmts[2], "CREATE OR REPLACE FUNCTION ops.fn_test()")
		for _, s := range stmts {
			assert.NotContains(t, s, "DROP FUNCTION")
		}
	})
}

func TestLoadMigrationsAndIdempotentRun(t *testing.T) {
	fsys := fstest.MapFS{
		"001-init.sql": &fstest.MapFile{
			Data: []byte("-- +goose Up\nCREATE TABLE IF NOT EXISTS public.a (id INT);"),
		},
		"002-alpha.sql": &fstest.MapFile{
			Data: []byte("-- +goose Up\nALTER TABLE public.a ADD COLUMN b INT;"),
		},
		"002-beta.sql": &fstest.MapFile{
			Data: []byte("-- +goose Up\n-- +goose StatementBegin\nDO $$\nBEGIN\n  NULL;\nEND $$;\n-- +goose StatementEnd"),
		},
		"003-gamma.sql": &fstest.MapFile{
			Data: []byte("-- +goose Up\nCREATE INDEX IF NOT EXISTS idx_a ON public.a(id);"),
		},
	}

	migrations, err := loadMigrations(fsys)
	require.NoError(t, err)
	require.Len(t, migrations, 4)

	assert.Equal(t, int64(1), migrations[0].Version)
	assert.Equal(t, int64(2), migrations[1].Version)
	assert.Equal(t, "002-alpha.sql", migrations[1].Filename)
	assert.Equal(t, int64(3), migrations[2].Version)
	assert.Equal(t, "002-beta.sql", migrations[2].Filename)
	assert.Equal(t, int64(4), migrations[3].Version)

	db := &fakeSQLDB{
		applied:    map[int64]bool{1: true},
		stmtsByVer: map[int64][]string{},
	}

	require.NoError(t, run(context.Background(), db, fsys))
	assert.Equal(t, []int64{2, 3, 4}, db.appliedSeq)
	assert.Contains(t, strings.Join(db.execCalls, "\n"), "public.goose_db_version")

	// Second run is a clean no-op
	db.appliedSeq = nil
	require.NoError(t, run(context.Background(), db, fsys))
	assert.Empty(t, db.appliedSeq)
}
