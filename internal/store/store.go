package store

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"time"
)

type DB struct{ db *sql.DB }
type Invoice struct {
	ID         string `json:"id"`
	ClientName string `json:"name"`
	Amount     int    `json:"amount"`
	DueDate    string `json:"due_date"`
	Status     string `json:"status"`
	LineItems  string `json:"line_items"`
	Notes      string `json:"notes"`
	PaidAt     string `json:"paid_at"`
	CreatedAt  string `json:"created_at"`
}

func Open(d string) (*DB, error) {
	if err := os.MkdirAll(d, 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(d, "billfold.db")+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS invoices(id TEXT PRIMARY KEY,name TEXT NOT NULL,amount INTEGER DEFAULT 0,due_date TEXT DEFAULT '',status TEXT DEFAULT 'draft',line_items TEXT DEFAULT '[]',notes TEXT DEFAULT '',paid_at TEXT DEFAULT '',created_at TEXT DEFAULT(datetime('now')))`)
	db.Exec(`CREATE TABLE IF NOT EXISTS extras(
	resource TEXT NOT NULL,
	record_id TEXT NOT NULL,
	data TEXT NOT NULL DEFAULT '{}',
	PRIMARY KEY(resource, record_id)
)`)
	return &DB{db: db}, nil
}
func (d *DB) Close() error { return d.db.Close() }
func genID() string        { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string          { return time.Now().UTC().Format(time.RFC3339) }
func (d *DB) Create(e *Invoice) error {
	e.ID = genID()
	e.CreatedAt = now()
	_, err := d.db.Exec(`INSERT INTO invoices(id,name,amount,due_date,status,line_items,notes,paid_at,created_at)VALUES(?,?,?,?,?,?,?,?,?)`, e.ID, e.ClientName, e.Amount, e.DueDate, e.Status, e.LineItems, e.Notes, e.PaidAt, e.CreatedAt)
	return err
}
func (d *DB) Get(id string) *Invoice {
	var e Invoice
	if d.db.QueryRow(`SELECT id,name,amount,due_date,status,line_items,notes,paid_at,created_at FROM invoices WHERE id=?`, id).Scan(&e.ID, &e.ClientName, &e.Amount, &e.DueDate, &e.Status, &e.LineItems, &e.Notes, &e.PaidAt, &e.CreatedAt) != nil {
		return nil
	}
	return &e
}
func (d *DB) List() []Invoice {
	rows, _ := d.db.Query(`SELECT id,name,amount,due_date,status,line_items,notes,paid_at,created_at FROM invoices ORDER BY created_at DESC`)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var o []Invoice
	for rows.Next() {
		var e Invoice
		rows.Scan(&e.ID, &e.ClientName, &e.Amount, &e.DueDate, &e.Status, &e.LineItems, &e.Notes, &e.PaidAt, &e.CreatedAt)
		o = append(o, e)
	}
	return o
}
func (d *DB) Update(e *Invoice) error {
	_, err := d.db.Exec(`UPDATE invoices SET name=?,amount=?,due_date=?,status=?,line_items=?,notes=?,paid_at=? WHERE id=?`, e.ClientName, e.Amount, e.DueDate, e.Status, e.LineItems, e.Notes, e.PaidAt, e.ID)
	return err
}
func (d *DB) Delete(id string) error {
	_, err := d.db.Exec(`DELETE FROM invoices WHERE id=?`, id)
	return err
}
func (d *DB) Count() int {
	var n int
	d.db.QueryRow(`SELECT COUNT(*) FROM invoices`).Scan(&n)
	return n
}

func (d *DB) Search(q string, filters map[string]string) []Invoice {
	where := "1=1"
	args := []any{}
	if q != "" {
		where += " AND (name LIKE ?)"
		args = append(args, "%"+q+"%")
	}
	if v, ok := filters["status"]; ok && v != "" {
		where += " AND status=?"
		args = append(args, v)
	}
	rows, _ := d.db.Query(`SELECT id,name,amount,due_date,status,line_items,notes,paid_at,created_at FROM invoices WHERE `+where+` ORDER BY created_at DESC`, args...)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var o []Invoice
	for rows.Next() {
		var e Invoice
		rows.Scan(&e.ID, &e.ClientName, &e.Amount, &e.DueDate, &e.Status, &e.LineItems, &e.Notes, &e.PaidAt, &e.CreatedAt)
		o = append(o, e)
	}
	return o
}

func (d *DB) Stats() map[string]any {
	m := map[string]any{"total": d.Count()}
	rows, _ := d.db.Query(`SELECT status,COUNT(*) FROM invoices GROUP BY status`)
	if rows != nil {
		defer rows.Close()
		by := map[string]int{}
		for rows.Next() {
			var s string
			var c int
			rows.Scan(&s, &c)
			by[s] = c
		}
		m["by_status"] = by
	}
	return m
}

// ─── Extras: generic key-value storage for personalization custom fields ───

func (d *DB) GetExtras(resource, recordID string) string {
	var data string
	err := d.db.QueryRow(
		`SELECT data FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	).Scan(&data)
	if err != nil || data == "" {
		return "{}"
	}
	return data
}

func (d *DB) SetExtras(resource, recordID, data string) error {
	if data == "" {
		data = "{}"
	}
	_, err := d.db.Exec(
		`INSERT INTO extras(resource, record_id, data) VALUES(?, ?, ?)
		 ON CONFLICT(resource, record_id) DO UPDATE SET data=excluded.data`,
		resource, recordID, data,
	)
	return err
}

func (d *DB) DeleteExtras(resource, recordID string) error {
	_, err := d.db.Exec(
		`DELETE FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	)
	return err
}

func (d *DB) AllExtras(resource string) map[string]string {
	out := make(map[string]string)
	rows, _ := d.db.Query(
		`SELECT record_id, data FROM extras WHERE resource=?`,
		resource,
	)
	if rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, data string
		rows.Scan(&id, &data)
		out[id] = data
	}
	return out
}
