package utility_party

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Party struct {
	ID         int64
	GuildID    string
	ChannelID  string
	MessageID  string
	HostID     string
	Game       string
	MaxPlayers int
	TargetTime string
	Note       string
	CreatedAt  int64
	ExpiresAt  int64
	Status     string // 'open', 'full', 'expired', 'cancelled'
}

type PartyMember struct {
	PartyID  int64
	UserID   string
	Role     string // 'main', 'sub'
	JoinedAt int64
}

type repository struct {
	db *sql.DB
}

func (r *repository) migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS utility_parties (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    message_id TEXT NOT NULL DEFAULT '',
    host_id TEXT NOT NULL,
    game TEXT NOT NULL,
    max_players INTEGER NOT NULL DEFAULT 5,
    target_time TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'open'
);
CREATE INDEX IF NOT EXISTS idx_utility_parties_msg ON utility_parties(message_id);
CREATE INDEX IF NOT EXISTS idx_utility_parties_status ON utility_parties(status, expires_at);

CREATE TABLE IF NOT EXISTS utility_party_members (
    party_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'main',
    joined_at INTEGER NOT NULL,
    PRIMARY KEY (party_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_utility_party_members_party ON utility_party_members(party_id, role, joined_at);
`)
	return err
}

func (r *repository) createParty(ctx context.Context, p *Party) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
INSERT INTO utility_parties (guild_id, channel_id, message_id, host_id, game, max_players, target_time, note, created_at, expires_at, status)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.GuildID, p.ChannelID, p.MessageID, p.HostID, p.Game, p.MaxPlayers, p.TargetTime, p.Note, p.CreatedAt, p.ExpiresAt, p.Status)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	p.ID = id

	// Automatically add host as the first main player
	_, err = r.db.ExecContext(ctx, `
INSERT INTO utility_party_members (party_id, user_id, role, joined_at)
VALUES (?, ?, 'main', ?)`, id, p.HostID, p.CreatedAt)
	return id, err
}

func (r *repository) setMessageID(ctx context.Context, partyID int64, messageID string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE utility_parties SET message_id=? WHERE id=?", messageID, partyID)
	return err
}

func (r *repository) getParty(ctx context.Context, partyID int64) (*Party, error) {
	var p Party
	err := r.db.QueryRowContext(ctx, `
SELECT id, guild_id, channel_id, message_id, host_id, game, max_players, target_time, note, created_at, expires_at, status
FROM utility_parties WHERE id=?`, partyID).Scan(
		&p.ID, &p.GuildID, &p.ChannelID, &p.MessageID, &p.HostID, &p.Game, &p.MaxPlayers, &p.TargetTime, &p.Note, &p.CreatedAt, &p.ExpiresAt, &p.Status,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) getPartyByMessageID(ctx context.Context, messageID string) (*Party, error) {
	var p Party
	err := r.db.QueryRowContext(ctx, `
SELECT id, guild_id, channel_id, message_id, host_id, game, max_players, target_time, note, created_at, expires_at, status
FROM utility_parties WHERE message_id=?`, messageID).Scan(
		&p.ID, &p.GuildID, &p.ChannelID, &p.MessageID, &p.HostID, &p.Game, &p.MaxPlayers, &p.TargetTime, &p.Note, &p.CreatedAt, &p.ExpiresAt, &p.Status,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) getMembers(ctx context.Context, partyID int64) ([]PartyMember, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT party_id, user_id, role, joined_at
FROM utility_party_members
WHERE party_id=?
ORDER BY role ASC, joined_at ASC`, partyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []PartyMember
	for rows.Next() {
		var m PartyMember
		if err := rows.Scan(&m.PartyID, &m.UserID, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

type JoinResult struct {
	ActionTaken string // "joined_main", "joined_sub", "switched_to_main", "switched_to_sub", "already_in_role", "main_full", "party_closed"
	IsNowFull   bool
}

func (r *repository) join(ctx context.Context, partyID int64, userID, targetRole string) (JoinResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return JoinResult{}, err
	}
	defer tx.Rollback()

	var p Party
	err = tx.QueryRowContext(ctx, `
SELECT id, max_players, status
FROM utility_parties WHERE id=?`, partyID).Scan(&p.ID, &p.MaxPlayers, &p.Status)
	if err != nil {
		return JoinResult{}, err
	}

	if p.Status != "open" && p.Status != "full" {
		return JoinResult{ActionTaken: "party_closed"}, nil
	}

	var currentRole string
	err = tx.QueryRowContext(ctx, `
SELECT role FROM utility_party_members WHERE party_id=? AND user_id=?`, partyID, userID).Scan(&currentRole)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return JoinResult{}, err
	}

	if currentRole == targetRole {
		return JoinResult{ActionTaken: "already_in_role"}, nil
	}

	var mainCount int
	err = tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM utility_party_members WHERE party_id=? AND role='main'`, partyID).Scan(&mainCount)
	if err != nil {
		return JoinResult{}, err
	}

	now := time.Now().Unix()

	if targetRole == "main" {
		if mainCount >= p.MaxPlayers {
			return JoinResult{ActionTaken: "main_full"}, nil
		}

		if currentRole == "sub" {
			_, err = tx.ExecContext(ctx, "UPDATE utility_party_members SET role='main', joined_at=? WHERE party_id=? AND user_id=?", now, partyID, userID)
			if err != nil {
				return JoinResult{}, err
			}
		} else {
			_, err = tx.ExecContext(ctx, "INSERT INTO utility_party_members (party_id, user_id, role, joined_at) VALUES (?, ?, 'main', ?)", partyID, userID, now)
			if err != nil {
				return JoinResult{}, err
			}
		}
		mainCount++
	} else { // targetRole == "sub"
		if currentRole == "main" {
			_, err = tx.ExecContext(ctx, "UPDATE utility_party_members SET role='sub', joined_at=? WHERE party_id=? AND user_id=?", now, partyID, userID)
			if err != nil {
				return JoinResult{}, err
			}
			mainCount--
		} else {
			_, err = tx.ExecContext(ctx, "INSERT INTO utility_party_members (party_id, user_id, role, joined_at) VALUES (?, ?, 'sub', ?)", partyID, userID, now)
			if err != nil {
				return JoinResult{}, err
			}
		}
	}

	isNowFull := mainCount >= p.MaxPlayers
	newStatus := "open"
	if isNowFull {
		newStatus = "full"
	}

	_, err = tx.ExecContext(ctx, "UPDATE utility_parties SET status=? WHERE id=?", newStatus, partyID)
	if err != nil {
		return JoinResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return JoinResult{}, err
	}

	action := "joined_" + targetRole
	if currentRole != "" {
		action = "switched_to_" + targetRole
	}

	return JoinResult{ActionTaken: action, IsNowFull: isNowFull}, nil
}

type LeaveResult struct {
	WasMember      bool
	PromotedUserID string // If a sub member was automatically promoted to main
	IsNowOpen      bool
	PartyCancelled bool
}

func (r *repository) leave(ctx context.Context, partyID int64, userID string) (LeaveResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return LeaveResult{}, err
	}
	defer tx.Rollback()

	var p Party
	err = tx.QueryRowContext(ctx, `
SELECT id, host_id, max_players, status
FROM utility_parties WHERE id=?`, partyID).Scan(&p.ID, &p.HostID, &p.MaxPlayers, &p.Status)
	if err != nil {
		return LeaveResult{}, err
	}

	var currentRole string
	err = tx.QueryRowContext(ctx, `
SELECT role FROM utility_party_members WHERE party_id=? AND user_id=?`, partyID, userID).Scan(&currentRole)
	if errors.Is(err, sql.ErrNoRows) {
		return LeaveResult{WasMember: false}, nil
	}
	if err != nil {
		return LeaveResult{}, err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM utility_party_members WHERE party_id=? AND user_id=?", partyID, userID)
	if err != nil {
		return LeaveResult{}, err
	}

	promotedUser := ""
	if currentRole == "main" {
		// Try to promote the earliest sub member
		var subUserID string
		err = tx.QueryRowContext(ctx, `
SELECT user_id FROM utility_party_members WHERE party_id=? AND role='sub' ORDER BY joined_at ASC LIMIT 1`, partyID).Scan(&subUserID)
		if err == nil && subUserID != "" {
			_, err = tx.ExecContext(ctx, "UPDATE utility_party_members SET role='main' WHERE party_id=? AND user_id=?", partyID, subUserID)
			if err == nil {
				promotedUser = subUserID
			}
		}
	}

	var remainingCount, mainCount int
	_ = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM utility_party_members WHERE party_id=?", partyID).Scan(&remainingCount)
	_ = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM utility_party_members WHERE party_id=? AND role='main'", partyID).Scan(&mainCount)

	if remainingCount == 0 {
		// All members left, cancel party
		_, _ = tx.ExecContext(ctx, "UPDATE utility_parties SET status='cancelled' WHERE id=?", partyID)
		return LeaveResult{WasMember: true, PartyCancelled: true}, tx.Commit()
	}

	// If host left, assign next main member as new host
	if userID == p.HostID {
		var newHostID string
		_ = tx.QueryRowContext(ctx, "SELECT user_id FROM utility_party_members WHERE party_id=? AND role='main' ORDER BY joined_at ASC LIMIT 1", partyID).Scan(&newHostID)
		if newHostID != "" {
			_, _ = tx.ExecContext(ctx, "UPDATE utility_parties SET host_id=? WHERE id=?", newHostID, partyID)
		}
	}

	isNowOpen := mainCount < p.MaxPlayers
	newStatus := "open"
	if mainCount >= p.MaxPlayers {
		newStatus = "full"
	}
	_, _ = tx.ExecContext(ctx, "UPDATE utility_parties SET status=? WHERE id=?", newStatus, partyID)

	if err := tx.Commit(); err != nil {
		return LeaveResult{}, err
	}

	return LeaveResult{
		WasMember:      true,
		PromotedUserID: promotedUser,
		IsNowOpen:      isNowOpen,
	}, nil
}

func (r *repository) cancel(ctx context.Context, partyID int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE utility_parties SET status='cancelled' WHERE id=?", partyID)
	return err
}

func (r *repository) getExpiredParties(ctx context.Context, nowUnix int64) ([]Party, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, guild_id, channel_id, message_id, host_id, game, max_players, target_time, note, created_at, expires_at, status
FROM utility_parties
WHERE status IN ('open', 'full') AND expires_at > 0 AND expires_at <= ?`, nowUnix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parties []Party
	for rows.Next() {
		var p Party
		if err := rows.Scan(
			&p.ID, &p.GuildID, &p.ChannelID, &p.MessageID, &p.HostID, &p.Game, &p.MaxPlayers, &p.TargetTime, &p.Note, &p.CreatedAt, &p.ExpiresAt, &p.Status,
		); err != nil {
			return nil, err
		}
		parties = append(parties, p)
	}
	return parties, nil
}

func (r *repository) markExpired(ctx context.Context, partyID int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE utility_parties SET status='expired' WHERE id=?", partyID)
	return err
}
