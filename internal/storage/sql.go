package storage

import "database/sql"

// SQL lets features own their tables using the shared, bounded connection pool.
// Features must not close this database; lifecycle ownership stays with main.
func (s *Store) SQL() *sql.DB {
	if s == nil {
		return nil
	}
	return s.db
}
