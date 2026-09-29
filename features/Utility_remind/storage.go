package remind

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type repository struct{ db *sql.DB }
type job struct {
	ID, Guild, User, Message, Status string
	Due, Next, Lease                 int64
	Attempts                         int
}

func (r *repository) migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS utility_remind_jobs (
 id TEXT PRIMARY KEY, guild TEXT NOT NULL, user TEXT NOT NULL, message TEXT NOT NULL,
 due INTEGER NOT NULL, next_attempt INTEGER NOT NULL, attempts INTEGER NOT NULL DEFAULT 0,
 status TEXT NOT NULL DEFAULT 'pending', lease_until INTEGER NOT NULL DEFAULT 0);
 CREATE INDEX IF NOT EXISTS utility_remind_due ON utility_remind_jobs(guild,status,next_attempt);
 CREATE INDEX IF NOT EXISTS utility_remind_owner ON utility_remind_jobs(guild,user);`)
	return err
}

var errLimit = errors.New("เก็บได้ไม่เกิน 20 รายการต่อคน และ 1000 รายการต่อเซิร์ฟเวอร์ กรุณาลบรายการเก่าก่อน")

func (r *repository) add(ctx context.Context, j job) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var own, total int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM utility_remind_jobs WHERE guild=? AND user=?", j.Guild, j.User).Scan(&own); err != nil {
		return err
	}
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM utility_remind_jobs WHERE guild=?", j.Guild).Scan(&total); err != nil {
		return err
	}
	if own >= 20 || total >= 1000 {
		return errLimit
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO utility_remind_jobs(id,guild,user,message,due,next_attempt) VALUES(?,?,?,?,?,?)", j.ID, j.Guild, j.User, j.Message, j.Due, j.Due)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *repository) list(ctx context.Context, guild, user string) ([]job, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id,message,due,status,attempts FROM utility_remind_jobs WHERE guild=? AND user=? ORDER BY due LIMIT 20", guild, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []job{}
	for rows.Next() {
		var j job
		if err = rows.Scan(&j.ID, &j.Message, &j.Due, &j.Status, &j.Attempts); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}
func (r *repository) cancel(ctx context.Context, guild, user, id string) (bool, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM utility_remind_jobs WHERE id=? AND guild=? AND user=? AND status IN ('pending','failed')", id, guild, user)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}
func (r *repository) claim(ctx context.Context, guild string, now int64) (*job, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var j job
	var query string
	var args []any
	if guild != "" {
		query = `SELECT id,guild,user,message,due,next_attempt,attempts FROM utility_remind_jobs WHERE guild=? AND ((status='pending' AND next_attempt<=?) OR (status='processing' AND lease_until<=?)) ORDER BY due LIMIT 1`
		args = []any{guild, now, now}
	} else {
		query = `SELECT id,guild,user,message,due,next_attempt,attempts FROM utility_remind_jobs WHERE ((status='pending' AND next_attempt<=?) OR (status='processing' AND lease_until<=?)) ORDER BY due LIMIT 1`
		args = []any{now, now}
	}
	err = tx.QueryRowContext(ctx, query, args...).Scan(&j.ID, &j.Guild, &j.User, &j.Message, &j.Due, &j.Next, &j.Attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, "UPDATE utility_remind_jobs SET status='processing',lease_until=?,attempts=attempts+1 WHERE id=?", now+60, j.ID)
	if err != nil {
		return nil, err
	}
	j.Attempts++
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &j, nil
}
func (r *repository) complete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM utility_remind_jobs WHERE id=?", id)
	return err
}
func (r *repository) retry(ctx context.Context, j job, now int64, permanent bool) error {
	status := "pending"
	if permanent || j.Attempts >= 5 {
		status = "failed"
	}
	delay := int64(60)
	for k := 1; k < j.Attempts && k < 5; k++ {
		delay *= 2
	}
	_, err := r.db.ExecContext(ctx, "UPDATE utility_remind_jobs SET status=?,next_attempt=?,lease_until=0 WHERE id=?", status, now+delay, j.ID)
	if err != nil {
		return fmt.Errorf("persist retry: %w", err)
	}
	return nil
}
