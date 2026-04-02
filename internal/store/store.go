package store
import ("database/sql";"fmt";"os";"path/filepath";"time";_ "modernc.org/sqlite")
type DB struct{db *sql.DB}
type Wallet struct{
	ID string `json:"id"`
	Name string `json:"name"`
	Currency string `json:"currency"`
	Balance float64 `json:"balance"`
	Type string `json:"type"`
	Description string `json:"description"`
	CreatedAt string `json:"created_at"`
}
func Open(d string)(*DB,error){if err:=os.MkdirAll(d,0755);err!=nil{return nil,err};db,err:=sql.Open("sqlite",filepath.Join(d,"billfold.db")+"?_journal_mode=WAL&_busy_timeout=5000");if err!=nil{return nil,err}
db.Exec(`CREATE TABLE IF NOT EXISTS wallets(id TEXT PRIMARY KEY,name TEXT NOT NULL,currency TEXT DEFAULT 'USD',balance REAL DEFAULT 0,type TEXT DEFAULT 'personal',description TEXT DEFAULT '',created_at TEXT DEFAULT(datetime('now')))`)
return &DB{db:db},nil}
func(d *DB)Close()error{return d.db.Close()}
func genID()string{return fmt.Sprintf("%d",time.Now().UnixNano())}
func now()string{return time.Now().UTC().Format(time.RFC3339)}
func(d *DB)Create(e *Wallet)error{e.ID=genID();e.CreatedAt=now();_,err:=d.db.Exec(`INSERT INTO wallets(id,name,currency,balance,type,description,created_at)VALUES(?,?,?,?,?,?,?)`,e.ID,e.Name,e.Currency,e.Balance,e.Type,e.Description,e.CreatedAt);return err}
func(d *DB)Get(id string)*Wallet{var e Wallet;if d.db.QueryRow(`SELECT id,name,currency,balance,type,description,created_at FROM wallets WHERE id=?`,id).Scan(&e.ID,&e.Name,&e.Currency,&e.Balance,&e.Type,&e.Description,&e.CreatedAt)!=nil{return nil};return &e}
func(d *DB)List()[]Wallet{rows,_:=d.db.Query(`SELECT id,name,currency,balance,type,description,created_at FROM wallets ORDER BY created_at DESC`);if rows==nil{return nil};defer rows.Close();var o []Wallet;for rows.Next(){var e Wallet;rows.Scan(&e.ID,&e.Name,&e.Currency,&e.Balance,&e.Type,&e.Description,&e.CreatedAt);o=append(o,e)};return o}
func(d *DB)Delete(id string)error{_,err:=d.db.Exec(`DELETE FROM wallets WHERE id=?`,id);return err}
func(d *DB)Count()int{var n int;d.db.QueryRow(`SELECT COUNT(*) FROM wallets`).Scan(&n);return n}
