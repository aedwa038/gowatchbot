package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aedwa038/ps5watcherbot/scraper"
	"github.com/aedwa038/ps5watcherbot/util"
)

// initTCPConnectionPool initializes a TCP connection pool for a Cloud SQL
// instance of SQL Server.
func NewTCP(dbUser, dbPwd, dbPort, dbTcpHost, dbName string) (*sql.DB, error) {

	dbURI := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPwd, dbTcpHost, dbPort, dbName)

	// dbPool is the pool of database connections.
	dbPool, err := sql.Open("mysql", dbURI)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %v", err)
	}

	configureConnectionPool(dbPool)

	return dbPool, nil
}

// initSocketConnectionPool initializes a Unix socket connection pool for
// this is used for testing purposes with cloudsql proxy
func NewSocket(dbUser, dbPwd, instanceConnectionName, dbName string) (*sql.DB, error) {

	socketDir, isSet := os.LookupEnv("DB_SOCKET_DIR")
	if !isSet {
		socketDir = "/cloudsql"
	}

	dbURI := fmt.Sprintf("%s:%s@unix(/%s/%s)/%s?parseTime=true", dbUser, dbPwd, socketDir, instanceConnectionName, dbName)

	dbPool, err := sql.Open("mysql", dbURI)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %v", err)
	}

	configureConnectionPool(dbPool)

	return dbPool, nil
}

func configureConnectionPool(dbPool *sql.DB) {

	// Set maximum number of connections in idle connection pool.
	dbPool.SetMaxIdleConns(5)

	// Set maximum number of open connections to the database.
	dbPool.SetMaxOpenConns(7)

	// Set Maximum time (in seconds) that a connection can remain open.
	dbPool.SetConnMaxLifetime(1800)

}

// saveVote saves a vote passed as http.Request form data.
func SaveConfig(db *sql.DB, name, value string) error {

	if insForm, err := db.Prepare("UPDATE Config SET Value = ? WHERE Name = ?"); err != nil {
		return fmt.Errorf("DB.Exec: %v", err)
	} else {
		_, err := insForm.Exec(value, name)
		if err != nil {
			return err
		}
	}
	return nil
}

//GetConfig gets the db config with a given key
func GetConfig(db *sql.DB, key string) (string, error) {
	var value string
	query := fmt.Sprintf("SELECT Value FROM Config where Config.Name = '%s'", key)
	err := db.QueryRow(query).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("DB.Config: %v %s", err, query)
	}
	return value, nil
}

//CheckHash checks if a given hash exists in the cron table
func CheckHash(db *sql.DB, key string) (bool, error) {
	var value int
	query := fmt.Sprintf("SELECT count(*) FROM CronResults where CronResults.hash= '%s'", key)
	err := db.QueryRow(query).Scan(&value)
	if err != nil {
		return false, fmt.Errorf("DB.getCheckHash: %v %s", err, query)
	}
	return value > 0, nil
}

//SaveCronResults saves cron job results to database
func SaveCronResults(db *sql.DB, results []scraper.Status) error {

	if data, err := json.Marshal(results); err == nil {
		hash := util.Hash(string(data))
		found, err := CheckHash(db, hash)
		if err != nil {
			return fmt.Errorf("DB.Exec: %v", err)
		}
		if !found {
			if insForm, err := db.Prepare("INSERT INTO CronResults(data, hash, updated) VALUES(?, ?, NOW())"); err == nil {
				insForm.Exec(data, hash)
			} else {
				return fmt.Errorf("DB.Exec: %v", err)
			}
		}
	} else {
		return fmt.Errorf("Json marshalling %v", err)
	}
	return nil
}

// SaveAvailableResults save the avaialbe items in a table
func SaveAvailableResults(db *sql.DB, results []scraper.Status) error {
	if len(results) == 0 {
		return nil
	}
	lastdate := results[len(results)-1].Date.Format(util.DateTempate)
	fmt.Println(lastdate)
	if err := SaveConfig(db, util.CacheKey, lastdate); err != nil {
		return fmt.Errorf("DB.saveAvailableResults: %v", err)
	}

	if data, err := json.Marshal(results); err == nil {
		if insForm, err := db.Prepare("INSERT INTO AvailablePS5(data, updated) VALUES(?, NOW())"); err == nil {
			insForm.Exec(data)
		} else {
			return fmt.Errorf("DB.Exec: %v", err)
		}
	} else {
		return fmt.Errorf("Json marshalling %v", err)
	}
	return nil
}
