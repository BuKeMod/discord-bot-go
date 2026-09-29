package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }
type Welcome struct{ Channel, Message string }

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
 	_, err = db.Exec(`PRAGMA busy_timeout=5000;
 PRAGMA journal_mode=WAL;
 CREATE TABLE IF NOT EXISTS welcome (guild TEXT PRIMARY KEY, channel TEXT NOT NULL, message TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS allowed_roles (guild TEXT NOT NULL, role TEXT NOT NULL, PRIMARY KEY(guild, role));
 CREATE TABLE IF NOT EXISTS log_channels (guild TEXT NOT NULL, log_type TEXT NOT NULL, channel TEXT NOT NULL, PRIMARY KEY(guild, log_type));
 CREATE TABLE IF NOT EXISTS warns (id TEXT PRIMARY KEY, guild TEXT NOT NULL, user TEXT NOT NULL, moderator TEXT NOT NULL, reason TEXT NOT NULL, created_at INTEGER NOT NULL);
 CREATE INDEX IF NOT EXISTS idx_warns_user ON warns(guild, user);
 CREATE TABLE IF NOT EXISTS ticket_config (guild TEXT PRIMARY KEY, category TEXT NOT NULL DEFAULT '', support_role TEXT NOT NULL DEFAULT '', ticket_counter INTEGER NOT NULL DEFAULT 0);
 CREATE TABLE IF NOT EXISTS tickets (channel_id TEXT PRIMARY KEY, guild TEXT NOT NULL, owner_id TEXT NOT NULL, ticket_num INTEGER NOT NULL, status TEXT NOT NULL DEFAULT 'open', created_at INTEGER NOT NULL);
 CREATE INDEX IF NOT EXISTS idx_tickets_owner ON tickets(guild, owner_id, status);
 CREATE TABLE IF NOT EXISTS guild_modules (guild TEXT NOT NULL, module TEXT NOT NULL, enabled INTEGER NOT NULL, PRIMARY KEY(guild, module));`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Welcome(ctx context.Context, guild string) (Welcome, error) {
	var w Welcome
	err := s.db.QueryRowContext(ctx, "SELECT channel,message FROM welcome WHERE guild=?", guild).Scan(&w.Channel, &w.Message)
	if err == sql.ErrNoRows {
		err = nil
	}
	return w, err
}
func (s *Store) SetWelcome(ctx context.Context, guild string, w Welcome) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO welcome VALUES(?,?,?) ON CONFLICT(guild) DO UPDATE SET channel=excluded.channel,message=excluded.message", guild, w.Channel, w.Message)
	return err
}
func (s *Store) Allow(ctx context.Context, guild, role string, allow bool) error {
	query := "DELETE FROM allowed_roles WHERE guild=? AND role=?"
	if allow {
		query = "INSERT OR IGNORE INTO allowed_roles VALUES(?,?)"
	}
	_, err := s.db.ExecContext(ctx, query, guild, role)
	return err
}
func (s *Store) Allowed(ctx context.Context, guild, role string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM allowed_roles WHERE guild=? AND role=?", guild, role).Scan(&count)
	return count > 0, err
}

func (s *Store) LogChannel(ctx context.Context, guild, logType string) (string, error) {
	var ch string
	err := s.db.QueryRowContext(ctx, "SELECT channel FROM log_channels WHERE guild=? AND log_type=?", guild, logType).Scan(&ch)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return ch, err
}

func (s *Store) SetLogChannel(ctx context.Context, guild, logType, channel string) error {
	if logType == "all" {
		for _, t := range []string{"voice", "message", "member", "mod", "ticket"} {
			if err := s.SetLogChannel(ctx, guild, t, channel); err != nil {
				return err
			}
		}
		return nil
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO log_channels(guild, log_type, channel) VALUES(?,?,?) ON CONFLICT(guild, log_type) DO UPDATE SET channel=excluded.channel", guild, logType, channel)
	return err
}

func (s *Store) DeleteLogChannel(ctx context.Context, guild, logType string) error {
	if logType == "all" {
		_, err := s.db.ExecContext(ctx, "DELETE FROM log_channels WHERE guild=?", guild)
		return err
	}
	_, err := s.db.ExecContext(ctx, "DELETE FROM log_channels WHERE guild=? AND log_type=?", guild, logType)
	return err
}

func (s *Store) AllLogChannels(ctx context.Context, guild string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT log_type, channel FROM log_channels WHERE guild=?", guild)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var t, ch string
		if err := rows.Scan(&t, &ch); err != nil {
			return nil, err
		}
		m[t] = ch
	}
	return m, rows.Err()
}

type Warn struct {
	ID        string
	Guild     string
	User      string
	Moderator string
	Reason    string
	CreatedAt int64
}

func (s *Store) AddWarn(ctx context.Context, id, guild, user, mod, reason string, createdAt int64) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO warns(id, guild, user, moderator, reason, created_at) VALUES(?,?,?,?,?,?)", id, guild, user, mod, reason, createdAt)
	return err
}

func (s *Store) ListWarns(ctx context.Context, guild, user string) ([]Warn, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, guild, user, moderator, reason, created_at FROM warns WHERE guild=? AND user=? ORDER BY created_at DESC", guild, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Warn
	for rows.Next() {
		var w Warn
		if err := rows.Scan(&w.ID, &w.Guild, &w.User, &w.Moderator, &w.Reason, &w.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func (s *Store) RemoveWarn(ctx context.Context, guild, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM warns WHERE guild=? AND id=?", guild, id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	return affected > 0, err
}

type TicketConfig struct {
	Category      string
	SupportRole   string
	TicketCounter int
}

func (s *Store) GetTicketConfig(ctx context.Context, guild string) (TicketConfig, error) {
	var cfg TicketConfig
	err := s.db.QueryRowContext(ctx, "SELECT category, support_role, ticket_counter FROM ticket_config WHERE guild=?", guild).Scan(&cfg.Category, &cfg.SupportRole, &cfg.TicketCounter)
	if err == sql.ErrNoRows {
		return TicketConfig{}, nil
	}
	return cfg, err
}

func (s *Store) SetTicketConfig(ctx context.Context, guild, category, supportRole string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO ticket_config(guild, category, support_role, ticket_counter)
VALUES(?,?,?, 0)
ON CONFLICT(guild) DO UPDATE SET category=excluded.category, support_role=excluded.support_role;`, guild, category, supportRole)
	return err
}

func (s *Store) NextTicketNumber(ctx context.Context, guild string) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	_, _ = tx.ExecContext(ctx, "INSERT OR IGNORE INTO ticket_config(guild, category, support_role, ticket_counter) VALUES(?, '', '', 0)", guild)
	var count int
	if err := tx.QueryRowContext(ctx, "SELECT ticket_counter FROM ticket_config WHERE guild=?", guild).Scan(&count); err != nil {
		return 0, err
	}
	count++
	if _, err := tx.ExecContext(ctx, "UPDATE ticket_config SET ticket_counter=? WHERE guild=?", count, guild); err != nil {
		return 0, err
	}
	return count, tx.Commit()
}

type TicketInfo struct {
	ChannelID string
	Guild     string
	OwnerID   string
	TicketNum int
	Status    string
	CreatedAt int64
}

func (s *Store) CreateTicket(ctx context.Context, channelID, guild, ownerID string, num int, createdAt int64) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO tickets(channel_id, guild, owner_id, ticket_num, status, created_at) VALUES(?,?,?,?, 'open', ?)", channelID, guild, ownerID, num, createdAt)
	return err
}

func (s *Store) GetActiveTicket(ctx context.Context, guild, ownerID string) (string, error) {
	var chID string
	err := s.db.QueryRowContext(ctx, "SELECT channel_id FROM tickets WHERE guild=? AND owner_id=? AND status='open'", guild, ownerID).Scan(&chID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return chID, err
}

func (s *Store) CloseTicket(ctx context.Context, channelID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE tickets SET status='closed' WHERE channel_id=?", channelID)
	return err
}

func (s *Store) GetTicket(ctx context.Context, channelID string) (*TicketInfo, error) {
	var t TicketInfo
	err := s.db.QueryRowContext(ctx, "SELECT channel_id, guild, owner_id, ticket_num, status, created_at FROM tickets WHERE channel_id=?", channelID).Scan(&t.ChannelID, &t.Guild, &t.OwnerID, &t.TicketNum, &t.Status, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// IsModuleEnabled checks whether a module is enabled for a guild.
// Default for all modules is true, except "roles" which is false by default.
func (s *Store) IsModuleEnabled(ctx context.Context, guild, module string) bool {
	if s == nil || s.db == nil || guild == "" {
		return true
	}
	var enabled int
	err := s.db.QueryRowContext(ctx, "SELECT enabled FROM guild_modules WHERE guild=? AND module=?", guild, module).Scan(&enabled)
	if err == sql.ErrNoRows {
		if module == "roles" {
			return false
		}
		return true
	}
	if err != nil {
		return true // fail open on error
	}
	return enabled == 1
}

// SetModuleEnabled updates the toggle status of a module for a guild.
func (s *Store) SetModuleEnabled(ctx context.Context, guild, module string, enabled bool) error {
	if s == nil || s.db == nil {
		return nil
	}
	val := 0
	if enabled {
		val = 1
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO guild_modules(guild, module, enabled) VALUES(?,?,?) ON CONFLICT(guild, module) DO UPDATE SET enabled=excluded.enabled", guild, module, val)
	return err
}

// GetGuildModules returns the enabled/disabled status of all toggleable modules for a guild.
func (s *Store) GetGuildModules(ctx context.Context, guild string) (map[string]bool, error) {
	allModules := []string{"moderation", "logs", "tickets", "party", "voice", "welcome", "roles", "fun", "utility"}
	res := make(map[string]bool)
	for _, m := range allModules {
		if m == "roles" {
			res[m] = false
		} else {
			res[m] = true
		}
	}
	if s == nil || s.db == nil || guild == "" {
		return res, nil
	}
	rows, err := s.db.QueryContext(ctx, "SELECT module, enabled FROM guild_modules WHERE guild=?", guild)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	for rows.Next() {
		var mod string
		var en int
		if err := rows.Scan(&mod, &en); err == nil {
			res[mod] = (en == 1)
		}
	}
	return res, rows.Err()
}


