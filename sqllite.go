package main

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	dbMutex    sync.Mutex
	dbPassword = "h4flow#1122" // 🔐 Strong password for encryption
	dbFile     = "secure_data.db"
)

// 🔹 Get an encrypted database connection
func getDBConnection() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFile+"?_pragma_key="+dbPassword+"&_pragma_cipher_page_size=4096")
	if err != nil {
		return nil, fmt.Errorf("failed to open encrypted database: %v", err)
	}

	// 🔐 Set the encryption key for SQLCipher
	_, err = db.Exec(fmt.Sprintf("PRAGMA key = '%s';", dbPassword))
	if err != nil {
		return nil, fmt.Errorf("failed to set encryption key: %v", err)
	}

	return db, nil
}

// 🔹 Initialize the database (Create tables)
func initializeDB() {
	db, err := getDBConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	schema := `
	CREATE TABLE IF NOT EXISTS attendance (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		checkinTime TEXT,
		checkoutTime TEXT,
		recordTime TEXT,
		worktimeMin REAL,
		idleTime REAL,
		workingTime REAL,
		date TEXT,
		email TEXT,
		machineID TEXT,
		status TEXT,
		type TEXT,
		ip TEXT
	);

	CREATE TABLE IF NOT EXISTS idle (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		start TEXT,
		end TEXT,
		durationMinutes REAL,
		duration REAL,
		date TEXT,
		email TEXT,
		machineID TEXT,
		status TEXT,
		type TEXT,
		ip TEXT
	);

	CREATE TABLE IF NOT EXISTS process (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		pid TEXT,
		name TEXT,
		status TEXT,
		user TEXT,
		cpu TEXT,
		email TEXT,
		type TEXT,
		date TEXT,
		timestamp TEXT,
		timeSpent TEXT,
		ip TEXT
	);`

	_, err = db.Exec(schema)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}
}

// 🔹 Structs for storing data
type AttendanceRecord struct {
	ID           int
	CheckinTime  string
	CheckoutTime string
	RecordTime   string
	WorktimeMin  float64
	IdleTime     float64
	WorkingTime  float64
	Date         string
	Email        string
	MachineID    string
	Status       string
	Type         string
	IP           string
}

type IdleRecord struct {
	ID              int
	Start           string
	End             string
	DurationMinutes float64
	Duration        float64
	Date            string
	Email           string
	MachineID       string
	Status          string
	Type            string
	IP              string
}

type ProcessInfo struct {
	ID        int
	PID       string
	Name      string
	Status    string
	User      string
	CPU       string
	Email     string
	Type      string
	Date      string
	Timestamp string
	TimeSpent string
	IP        string
}

// 🔹 Insert an attendance record
func insertAttendanceRecord(record AttendanceRecord) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	db, err := getDBConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	query := `INSERT INTO attendance (checkinTime, checkoutTime, recordTime, worktimeMin, idleTime, workingTime, date, email, machineID, status, type, ip) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = db.Exec(query, record.CheckinTime, record.CheckoutTime, record.RecordTime,
		record.WorktimeMin, record.IdleTime, record.WorkingTime, record.Date, record.Email,
		record.MachineID, record.Status, record.Type, record.IP)
	if err != nil {
		return fmt.Errorf("failed to insert attendance record: %v", err)
	}
	return nil
}

// 🔹 Retrieve all attendance records
func getAttendanceRecords() ([]AttendanceRecord, error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	db, err := getDBConnection()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM attendance")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve attendance records: %v", err)
	}
	defer rows.Close()

	var records []AttendanceRecord
	for rows.Next() {
		var record AttendanceRecord
		if err := rows.Scan(&record.ID, &record.CheckinTime, &record.CheckoutTime, &record.RecordTime,
			&record.WorktimeMin, &record.IdleTime, &record.WorkingTime, &record.Date, &record.Email,
			&record.MachineID, &record.Status, &record.Type, &record.IP); err != nil {
			return nil, fmt.Errorf("failed to scan record: %v", err)
		}
		records = append(records, record)
	}
	// deleteAttendanceRecords()
	return records, nil
}

// 🔹 Delete all attendance records
func deleteAttendanceRecords() error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	db, err := getDBConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM attendance")
	if err != nil {
		return fmt.Errorf("failed to delete attendance records: %v", err)
	}
	return nil
}
