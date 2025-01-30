package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/reader"
	"github.com/xitongsys/parquet-go/writer"
)

var fileMutex sync.Mutex

type AttendanceRecord struct {
	ID           string  `parquet:"name=_id, type=BYTE_ARRAY, convertedtype=UTF8"`
	CheckinTime  string  `parquet:"name=checkinTime, type=BYTE_ARRAY, convertedtype=UTF8"`
	CheckoutTime string  `parquet:"name=checkoutTime, type=BYTE_ARRAY, convertedtype=UTF8"`
	RecordTime   string  `parquet:"name=recordTime, type=BYTE_ARRAY, convertedtype=UTF8"`
	WorktimeMin  float64 `parquet:"name=worktimeMin, type=DOUBLE"`
	IdleTime     float64 `parquet:"name=idleTime, type=DOUBLE"`
	WorkingTime  float64 `parquet:"name=workingTime, type=DOUBLE"`
	Date         string  `parquet:"name=date, type=BYTE_ARRAY, convertedtype=UTF8"`
	Email        string  `parquet:"name=email, type=BYTE_ARRAY, convertedtype=UTF8"`
	MachineID    string  `parquet:"name=machineID, type=BYTE_ARRAY, convertedtype=UTF8"`
	Status       string  `parquet:"name=status, type=BYTE_ARRAY, convertedtype=UTF8"`
	Type         string  `parquet:"name=type, type=BYTE_ARRAY, convertedtype=UTF8"`
	IP           string  `parquet:"name=ip, type=BYTE_ARRAY, convertedtype=UTF8"`
}
type IdleRecord struct {
	Start           string  `parquet:"name=start, type=BYTE_ARRAY, convertedtype=UTF8"`
	End             string  `parquet:"name=end, type=BYTE_ARRAY, convertedtype=UTF8"`
	DurationMinutes float64 `parquet:"name=durationMin, type=DOUBLE"`
	Duration        float64 `parquet:"name=durationTime, type=DOUBLE"`
	Date            string  `parquet:"name=date, type=BYTE_ARRAY, convertedtype=UTF8"`
	Email           string  `parquet:"name=email, type=BYTE_ARRAY, convertedtype=UTF8"`
	MachineID       string  `parquet:"name=machineID, type=BYTE_ARRAY, convertedtype=UTF8"`
	Status          string  `parquet:"name=status, type=BYTE_ARRAY, convertedtype=UTF8"`
	Type            string  `parquet:"name=type, type=BYTE_ARRAY, convertedtype=UTF8"`
	IP              string  `parquet:"name=ip, type=BYTE_ARRAY, convertedtype=UTF8"`
}

type ProcessInfo struct {
	PID       string `parquet:"name=pid,type=BYTE_ARRAY, convertedtype=UTF8"`
	NAME      string `parquet:"name=name, type=BYTE_ARRAY, convertedtype=UTF8"`
	STATUS    string `parquet:"name=status, type=BYTE_ARRAY, convertedtype=UTF8"`
	USER      string `parquet:"name=user, type=BYTE_ARRAY, convertedtype=UTF8"`
	CPU       string `parquet:"name=cpu, type=BYTE_ARRAY, convertedtype=UTF8"` // Formatted percentage as a string
	Email     string `parquet:"name=email, type=BYTE_ARRAY, convertedtype=UTF8"`
	Type      string `parquet:"name=type, type=BYTE_ARRAY, convertedtype=UTF8"`
	Date      string `parquet:"name=date,type=BYTE_ARRAY, convertedtype=UTF8"`
	Timestamp string `parquet:"name=timestamp,type=BYTE_ARRAY, convertedtype=UTF8"`
	TimeSpent string `parquet:"name=time_spent, type=BYTE_ARRAY, convertedtype=UTF8"`
	IP        string `parquet:"name=ip, type=BYTE_ARRAY, convertedtype=UTF8"`
}

func writeParquetFile[T any](filename string, records []T) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	// Open the Parquet file for reading (if it exists)
	fr, err := local.NewLocalFileReader(filename)
	var existingRecords []T
	if err == nil {
		// Create a Parquet reader
		pr, err := reader.NewParquetReader(fr, new(T), 4)
		if err != nil {
			log.Fatalf("Failed to create Parquet reader: %v", err)
		}
		defer pr.ReadStop()

		// Read existing records
		numRows := int(pr.GetNumRows())
		existingRecords = make([]T, numRows)
		if err = pr.Read(&existingRecords); err != nil {
			log.Fatalf("Read error: %v", err)
		}
	}

	// Append the new records to the existing records
	combinedRecords := append(existingRecords, records...)
	if len(combinedRecords) == 0 {
		return
	}
	// Create a new Parquet file (or overwrite the existing one)
	fw, err := local.NewLocalFileWriter(filename)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer fw.Close()

	// Create a Parquet writer
	pw, err := writer.NewParquetWriter(fw, new(T), 4)
	if err != nil {
		log.Fatalf("Failed to create Parquet writer: %v", err)
	}
	defer pw.WriteStop()

	// Write the combined records to the Parquet file
	for _, record := range combinedRecords {
		if err = pw.Write(record); err != nil {
			log.Fatalf("Write error: %v", err)
		}
	}
	fmt.Printf("Data successfully written to %s\n", filename)
}

// Read data from a Parquet file and append to a []map[string]interface{}

func flushIdleRecords(filename string) ([]interface{}, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	// Open the Parquet file for reading
	fr, err := local.NewLocalFileReader(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer fr.Close()

	// Create a Parquet reader
	pr, err := reader.NewParquetReader(fr, new(IdleRecord), 4)
	if err != nil {
		return nil, fmt.Errorf("failed to create Parquet reader: %v", err)
	}
	defer pr.ReadStop()

	// Read data from the Parquet file
	numRows := int(pr.GetNumRows())
	records := make([]IdleRecord, numRows)
	if err = pr.Read(&records); err != nil {
		return nil, fmt.Errorf("read error: %v", err)
	}

	// Initialize a slice of maps to store the data
	var result []interface{}
	jsonData, err := json.Marshal(records)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %v", err)
	}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %v", err)
	}
	err = sendPostRequest(result, "idle")
	if err == nil {
		fmt.Println("err", err)
		clearParquetFile("idle.parquet")
	}
	return result, nil
}
func flushAttendanceRecords(filename string) ([]interface{}, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	// Open the Parquet file for reading
	fr, err := local.NewLocalFileReader(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer fr.Close()

	// Create a Parquet reader
	pr, err := reader.NewParquetReader(fr, new(AttendanceRecord), 4)
	if err != nil {
		return nil, fmt.Errorf("failed to create Parquet reader: %v", err)
	}
	defer pr.ReadStop()

	// Read data from the Parquet file
	numRows := int(pr.GetNumRows())
	records := make([]AttendanceRecord, numRows)
	if err = pr.Read(&records); err != nil {
		return nil, fmt.Errorf("read error: %v", err)
	}

	// Initialize a slice of maps to store the data
	var result []interface{}
	jsonData, err := json.Marshal(records)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %v", err)
	}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %v", err)
	}
	err = sendPostRequest(result, "idle")
	if err == nil {
		clearParquetFile("attendance.parquet")
	}
	clearParquetFile(filename)
	return result, nil
}
func flushProcessRecords(filename string) ([]map[string]interface{}, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	// Open the Parquet file for reading
	fr, err := local.NewLocalFileReader(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer fr.Close()

	// Create a Parquet reader
	pr, err := reader.NewParquetReader(fr, new(ProcessInfo), 4)
	if err != nil {
		return nil, fmt.Errorf("failed to create Parquet reader: %v", err)
	}
	defer pr.ReadStop()

	// Read data from the Parquet file
	numRows := int(pr.GetNumRows())
	records := make([]ProcessInfo, numRows)
	if err = pr.Read(&records); err != nil {
		return nil, fmt.Errorf("read error: %v", err)
	}

	// Initialize a slice of maps to store the data
	var result []map[string]interface{}
	jsonData, err := json.Marshal(records)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %v", err)
	}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %v", err)
	}
	err = sendProcess(result)
	if err == nil {
		fmt.Println("err", err)
		clearParquetFile("process.parquet")
	}
	clearParquetFile(filename)
	return result, nil
}
func clearParquetFile(filename string) {
	fileMutex.Lock()
	defer fileMutex.Unlock()
	// Create a new empty Parquet file with the same schema
	fw, err := local.NewLocalFileWriter(filename)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer fw.Close()

	// Create a Parquet writer
	pw, err := writer.NewParquetWriter(fw, new(AttendanceRecord), 4)
	if err != nil {
		log.Fatalf("Failed to create Parquet writer: %v", err)
	}
	defer pw.WriteStop()

	// Write no records to the file
	fmt.Printf("Parquet file %s has been cleared.\n", filename)
}
