package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/lib/pq"
)

type DBConfig struct {
	User     string
	Password string
	DBName   string
}

type School struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases"`
}

func SetDBConfig(config *DBConfig, key string, val string) {
	switch key {
	case "DB_USER":
		config.User = val
	case "DB_PASSWORD":
		config.Password = val
	case "DB_NAME":
		config.DBName = val
	}
}

func LoadDBConfig(filename string) (DBConfig, error) {
	config := DBConfig{}
	file, err := os.Open(filename)

	if err != nil {
		return config, fmt.Errorf("readfile %s, error occured: %s", filename, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		SetDBConfig(&config, key, value)
	}

	if err := scanner.Err(); err != nil {
		return config, err
	}
	return config, err
}

func InitDB() (*sql.DB, error) {
	config, err := LoadDBConfig(".env")

	if err != nil {
		fmt.Printf("File may not exist \".env\", error :%s\n", err)
	}
	dbName := "schools_db"
	dbUser := "admin"
	dbPassword := "secret"
	dbPort := "5432"

	if config.DBName != "" {
		dbName = config.DBName
	}
	if config.User != "" {
		dbUser = config.User
	}
	if config.Password != "" {
		dbPassword = config.Password
	}

	connSetting := fmt.Sprintf("user=%s password=%s dbname=%s port=%s sslmode=disable", dbUser, dbPassword, dbName, dbPort)

	db, err := sql.Open("postgres", connSetting)

	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("DB connection failed: %v", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute)

	fmt.Println("Connect to db successfully")

	return db, nil
}

func InitCreateSchema(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schools (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL UNIQUE, -- unique key for name
		aliases TEXT[],            -- postgres Array Type
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err := db.Exec(query)

	if err != nil {
		return fmt.Errorf("query failed")
	}

	return nil
}

func SaveUniToDB(db *sql.DB, data map[string][]string) error {
	fmt.Printf("save the number of data: %d", len(data))
	start := time.Now()
	tx, err := db.Begin()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	//setup statement first, we can put the variables into it later
	stmt, err := tx.Prepare(`
		INSERT INTO schools (name, aliases)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE
		SET aliases = EXCLUDED.aliases; -- Repeated, update its alias
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for name, aliases := range data {
		// pq.Array() from lib/pq，go slice -> Postgres Array
		_, err := stmt.Exec(name, pq.Array(aliases))
		if err != nil {
			log.Printf("write failed [%s]: %v\n", name, err)
			continue
		}
	}

	// Submit commit
	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("Finished, time spent: %v\n", time.Since(start))

	return nil
}

func SearchSchoolsDB(db *sql.DB, name string) ([]School, error) {
	//TODO: Need to search name also in alias

	start := time.Now()

	query := `
		SELECT name, aliases
		FROM schools
		WHERE name ILIKE $1 OR array_to_string(aliases, ',') ILIKE $1
		LIMIT 10
	`
	name = "%" + name + "%"

	rows, err := db.Query(query, name)
	results := []School{}

	if err != nil {
		return nil, fmt.Errorf("query failed %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var school School
		err := rows.Scan(&school.Name, pq.Array(&school.Aliases))
		if err != nil {
			log.Printf("Read failed %v\n", err)
			return nil, err
		}
		results = append(results, school)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	fmt.Println("Time spent in search: ", time.Since(start))

	return results, nil

}
