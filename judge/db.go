package judge

import (
	_ "github.com/godror/godror"
	"github.com/jmoiron/sqlx"
)

const oracleDriverName = "godror"

func connectDB(connectionString string) (*sqlx.DB, error) {
	db, err := sqlx.Connect(oracleDriverName, connectionString)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
