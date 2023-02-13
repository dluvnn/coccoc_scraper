package tracker

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/genjidb/genji/driver"
)

type Tracker struct {
	db *sql.DB
}

type UsersRequests map[string]int // user_id -> number_requests
type row struct {
}

func (tk *Tracker) Init(db_instance string) error {
	db, err := sql.Open("genji", db_instance)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS requests(user_id TEXT, created_at INTEGER, nreq INTEGER);
	CREATE INDEX ON requests(user_id, created_at);
	`)
	if err != nil {
		return err
	}

	tk.db = db
	return nil
}

func (tk *Tracker) Close() error {
	return tk.db.Close()
}

func (tk *Tracker) Update(ctx context.Context, info map[string]int64) error {
	t := time.Now().Unix()

	tx, err := tk.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for user_id, nreq := range info {
		_, err := tx.ExecContext(ctx, `INSERT INTO requests(user_id, created_at, nreq) VALUES(?, ?, ?);`, user_id, t, nreq)
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (tk *Tracker) parseRow(r *sql.Row) (int64, error) {
	err := r.Err()
	if err != nil {
		return 0, err
	}

	var nreq sql.NullInt64
	err = r.Scan(&nreq)
	if err != nil {
		return 0, err
	}
	if nreq.Valid {
		return nreq.Int64, nil
	}
	return 0, nil
}

func (tk *Tracker) QueryOne(ctx context.Context, user_id string, from, to int64) (int64, error) {
	return tk.parseRow(tk.db.QueryRowContext(ctx, `SELECT SUM(nreq) AS NumberRequests FROM requests WHERE user_id = ? AND created_at >= ? AND created_at < ?;`, user_id, from, to))
}

func (tk *Tracker) QueryAll(ctx context.Context, from, to int64) (int64, error) {
	return tk.parseRow(tk.db.QueryRowContext(ctx, `SELECT SUM(nreq) AS NumberRequests FROM requests WHERE created_at >= ? and created_at < ?;`, from, to))
}
