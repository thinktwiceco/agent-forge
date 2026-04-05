package brain

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func brainNodeColumnExists(db *sql.DB, col string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(brain_nodes)`)
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, col) {
			return true, nil
		}
	}
	return false, rows.Err()
}

// ensureBrainNodeExtendedColumns adds title, description, distillation_reason, search_text if missing.
func (p *BrainPlugin) ensureBrainNodeExtendedColumns(ctx context.Context) error {
	alters := []struct {
		col string
		ddl string
	}{
		{"title", "ALTER TABLE brain_nodes ADD COLUMN title TEXT"},
		{"description", "ALTER TABLE brain_nodes ADD COLUMN description TEXT"},
		{"distillation_reason", "ALTER TABLE brain_nodes ADD COLUMN distillation_reason TEXT"},
		{"search_text", "ALTER TABLE brain_nodes ADD COLUMN search_text TEXT NOT NULL DEFAULT ''"},
	}
	for _, a := range alters {
		ok, err := brainNodeColumnExists(p.db, a.col)
		if err != nil {
			return err
		}
		if ok {
			continue
		}
		if _, err := p.db.ExecContext(ctx, a.ddl); err != nil {
			return fmt.Errorf("brain migrate add column %s: %w", a.col, err)
		}
	}
	return nil
}

func (p *BrainPlugin) backfillSearchText(ctx context.Context) error {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, content, COALESCE(title,''), COALESCE(description,''), COALESCE(distillation_reason,'')
		FROM brain_nodes`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id, content, title, desc, reason string
		if err := rows.Scan(&id, &content, &title, &desc, &reason); err != nil {
			return err
		}
		st := buildSearchText(content, title, desc, reason)
		if _, err := p.db.ExecContext(ctx, `UPDATE brain_nodes SET search_text = ? WHERE id = ?`, st, id); err != nil {
			return err
		}
	}
	return rows.Err()
}

func ftsUsesSearchText(ctx context.Context, db *sql.DB) (bool, error) {
	var sqlStr sql.NullString
	err := db.QueryRowContext(ctx,
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='brain_fts'`).Scan(&sqlStr)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !sqlStr.Valid {
		return false, nil
	}
	s := strings.ToLower(sqlStr.String)
	return strings.Contains(s, "search_text") && strings.Contains(s, "fts5"), nil
}

func (p *BrainPlugin) dropFTSTriggersAndTable(ctx context.Context) {
	for _, name := range []string{"brain_fts_insert", "brain_fts_delete", "brain_fts_update"} {
		_, _ = p.db.ExecContext(ctx, "DROP TRIGGER IF EXISTS "+name)
	}
	_, _ = p.db.ExecContext(ctx, "DROP TABLE IF EXISTS brain_fts")
}

// backfillSearchTextIfNeeded fills search_text when empty (e.g. after ADD COLUMN).
func (p *BrainPlugin) backfillSearchTextIfNeeded(ctx context.Context) error {
	var n int
	if err := p.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM brain_nodes WHERE COALESCE(search_text, '') = ''`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	return p.backfillSearchText(ctx)
}

// migrateFTSToSearchText ensures FTS5 indexes search_text; recreates if an older content-only FTS exists.
func (p *BrainPlugin) migrateFTSToSearchText(ctx context.Context) error {
	ok, err := ftsUsesSearchText(ctx, p.db)
	if err != nil {
		return err
	}
	if ok {
		_, _ = p.db.ExecContext(ctx, "INSERT INTO brain_fts(brain_fts) VALUES('rebuild')")
		return nil
	}

	var ftsName string
	err = p.db.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name='brain_fts'`).Scan(&ftsName)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == nil && ftsName != "" {
		p.dropFTSTriggersAndTable(ctx)
	}

	ddls := []string{
		`CREATE VIRTUAL TABLE IF NOT EXISTS brain_fts
		 USING fts5(search_text, node_id UNINDEXED, content='brain_nodes', content_rowid='rowid')`,

		`CREATE TRIGGER IF NOT EXISTS brain_fts_insert
		 AFTER INSERT ON brain_nodes BEGIN
		     INSERT INTO brain_fts(rowid, search_text, node_id) VALUES (new.rowid, new.search_text, new.id);
		 END`,

		`CREATE TRIGGER IF NOT EXISTS brain_fts_delete
		 BEFORE DELETE ON brain_nodes BEGIN
		     INSERT INTO brain_fts(brain_fts, rowid, search_text, node_id)
		     VALUES ('delete', old.rowid, old.search_text, old.id);
		 END`,

		`CREATE TRIGGER IF NOT EXISTS brain_fts_update
		 AFTER UPDATE ON brain_nodes BEGIN
		     INSERT INTO brain_fts(brain_fts, rowid, search_text, node_id)
		     VALUES ('delete', old.rowid, old.search_text, old.id);
		     INSERT INTO brain_fts(rowid, search_text, node_id) VALUES (new.rowid, new.search_text, new.id);
		 END`,
	}
	for _, ddl := range ddls {
		if _, err := p.db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("brain fts5 search_text: %w", err)
		}
	}

	_, err = p.db.ExecContext(ctx, "INSERT INTO brain_fts(brain_fts) VALUES('rebuild')")
	return err
}
