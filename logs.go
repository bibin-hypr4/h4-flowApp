package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

var (
	logMutex sync.Mutex
	logFile  string
)

// // 🔹 Structs for storing data
// type AttendanceRecord struct {
// 	CheckinTime  string
// 	CheckoutTime string
// 	RecordTime   string
// 	WorktimeMin  float64
// 	IdleTime     float64
// 	WorkingTime  float64
// 	Date         string
// 	Email        string
// 	MachineID    string
// 	Status       string
// 	Type         string
// 	IP           string
// }

type IdleRecord struct {
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
	PID       string
	Name      string
	Status    string
	User      string
	Cpu       string
	Email     string
	Type      string
	Date      string
	Timestamp string
	TimeSpent string
	IP        string
}

// 🔹 Struct for storing attendance data
type AttendanceRecord struct {
	ID           int     `json:"id"`
	CheckinTime  string  `json:"CheckinTime"`
	CheckoutTime string  `json:"CheckoutTime"`
	RecordTime   string  `json:"RecordTime"`
	WorktimeMin  float64 `json:"WorktimeMin"`
	IdleTime     float64 `json:"IdleTime"`
	WorkingTime  float64 `json:"WorkingTime"`
	Date         string  `json:"Date"`
	Email        string  `json:"Email"`
	MachineID    string  `json:"MachineID"`
	Status       string  `json:"Status"`
	Type         string  `json:"Type"`
	IP           string  `json:"IP"`
}

// 🔹 Write an attendance record to the log file
func writeAttendanceRecord(record AttendanceRecord) error {
	logMutex.Lock()
	defer logMutex.Unlock()

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer file.Close()
	fmt.Println(logFile)
	recordJSON, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %v", err)
	}

	_, err = file.WriteString(string(recordJSON) + "\n")
	if err != nil {
		return fmt.Errorf("failed to write to log file: %v", err)
	}
	fmt.Println("done", record)
	return nil
}

// 🔹 Read all attendance records from the log file
func readAttendanceRecords() ([]AttendanceRecord, error) {
	logMutex.Lock()
	defer logMutex.Unlock()

	file, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []AttendanceRecord{}, nil // Return empty slice if file does not exist
		}
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}
	defer file.Close()

	var records []AttendanceRecord
	decoder := json.NewDecoder(file)

	for {
		var record AttendanceRecord
		if err := decoder.Decode(&record); err != nil {
			break
		}
		records = append(records, record)
	}

	return records, nil
}

// 🔹 Delete all attendance records by clearing the log file
func deleteAttendanceRecords() error {
	logMutex.Lock()
	defer logMutex.Unlock()

	err := os.WriteFile(logFile, []byte{}, 0644)
	if err != nil {
		return fmt.Errorf("failed to clear log file: %v", err)
	}
	return nil
}
