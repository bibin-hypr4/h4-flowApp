package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"fyne.io/systray"
)

func sendPostRequest(data interface{}, reqType string) error {
	var apiURL string

	if reqType == "idle" {
		apiURL = IDLE_URL
	} else if reqType == "attendance" {
		apiURL = ATTENDANCE_URL
	} else if apiURL == "" {
		return fmt.Errorf("API_URL environment variable not set")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshalling data to JSON: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating POST request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending POST request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("received non-OK response: %s", body)
	}

	return nil
}

func recordAttendance(recordType string, status, machineID string, checkinTime, checkoutTime time.Time, workingTime, dailyIdleTime time.Duration) {
	recordDetails := []AttendanceRecord{{
		Type:         recordType,
		Status:       status,
		Email:        userEmail,
		MachineID:    machineID,
		RecordTime:   time.Now().Format("2006-01-02T15:04:05.999999999-07:00"),
		CheckinTime:  checkinTime.Format("2006-01-02T15:04:05.999999999-07:00"),
		CheckoutTime: checkoutTime.Format("2006-01-02T15:04:05.999999999-07:00"),
		WorkingTime:  workingTime.Hours(),
		WorktimeMin:  workingTime.Minutes(),
		IdleTime:     dailyIdleTime.Minutes(),
		Date:         time.Now().Format("2006-01-02"),
		IP:           USER_IP,
	}}
	if status == "checked_in" {
		go func() {
			err := sendPostRequest(recordDetails, "attendance")
			if err != nil {
				writeAttendanceRecord(recordDetails)
			} else {
				processLogs()

			}
		}()
	} else {
		err := sendPostRequest(recordDetails, "attendance")
		if err != nil {
			writeAttendanceRecord(recordDetails)
		}
	}

}
func handleCrash(r interface{}) {
	if checkedIn {
		data := []AttendanceRecord{{
			Type:         "session",
			Status:       "checked_out",
			Email:        userEmail,
			MachineID:    machineID,
			RecordTime:   time.Now().Format("2006-01-02T15:04:05.999999999-07:00"),
			CheckinTime:  checkinTime.Format("2006-01-02T15:04:05.999999999-07:00"),
			CheckoutTime: time.Now().Format("2006-01-02T15:04:05.999999999-07:00"),
			WorkingTime:  workingTime.Hours(),
			IdleTime:     dailyIdleTime.Hours(),
			WorktimeMin:  workingTime.Minutes(),
			Date:         time.Now().Format("2006-01-02"),
			IP:           USER_IP,
		}, {
			Type:         "attendance",
			Status:       "checked_out",
			Email:        userEmail,
			MachineID:    machineID,
			RecordTime:   time.Now().Format("2006-01-02T15:04:05.999999999-07:00"),
			CheckinTime:  checkinTime.Format("2006-01-02T15:04:05.999999999-07:00"),
			CheckoutTime: time.Now().Format("2006-01-02T15:04:05.999999999-07:00"),
			WorkingTime:  workingTime.Hours(),
			IdleTime:     dailyIdleTime.Hours(),
			WorktimeMin:  workingTime.Minutes(),
			Date:         time.Now().Format("2006-01-02"),
			IP:           USER_IP,
		}}
		err := sendPostRequest(data, "attendance")
		if err != nil {
			writeAttendanceRecord(data)

		}
	}
	systray.Quit()
	log.Fatal("force quit or crash detected")
}
func getUserDetails(machineID string) (map[string]interface{}, error) {
	apiURL := GETUSER_URL
	if apiURL == "" {
		return nil, fmt.Errorf("API_URL environment variable not set")
	}
	filter := map[string]string{
		"machineID": machineID,
	}
	requestBody := map[string]interface{}{
		"filter": filter,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling data to JSON: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("error unmarshalling response body", err)

		return nil, fmt.Errorf("error creating POST request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Network error: %v", err)
		fmt.Println("error unmarshalling response body", err)

		return nil, fmt.Errorf("error sending POST request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {

		fmt.Println("error unmarshalling response body", err)

		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Println("error unmarshalling response body", err)
		return nil, fmt.Errorf("received non-OK response: %s", body)
	}

	var responseData map[string]interface{}
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		fmt.Println("error unmarshalling response body")
		return nil, fmt.Errorf("error unmarshalling response body: %v", err)
	}
	if _, exist := responseData["data"]; !exist {
		return nil, fmt.Errorf("no user found")
	}
	responceMap, exist := responseData["data"].(map[string]interface{})
	if len(responceMap) == 0 || !exist {
		return nil, fmt.Errorf("no user found")
	}
	return responceMap, nil
}
func FetchConfigDetails(apiURL string) (map[string]interface{}, error) {
	// Validate input
	if apiURL == "" {
		return nil, fmt.Errorf("API_URL environment variable not set")
	}

	// Create HTTP request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("Error creating GET request: %v\n", err)
		return nil, fmt.Errorf("error creating GET request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Network error: %v\n", err)
		return nil, fmt.Errorf("error sending GET request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v\n", err)
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("Non-OK HTTP status: %d, response: %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("received non-OK response: %s", body)
	}

	// Parse response JSON
	var responseData map[string]interface{}
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		log.Printf("Error unmarshalling response body: %v\n", err)
		return nil, fmt.Errorf("error unmarshalling response body: %v", err)
	}

	// Validate response structure
	data, exists := responseData["data"]
	if !exists {
		return nil, fmt.Errorf("missing 'data' field in response")
	}

	responseMap, ok := data.(map[string]interface{})
	if !ok || len(responseMap) == 0 {
		return nil, fmt.Errorf("no valid user data found")
	}

	// // Extract configuration details
	// if interval, exists := responseMap["processTimeInterval"]; exists {
	// 	if intervalFloat, ok := interval.(float64); ok {
	// 		processTimeInverval = time.Duration(intervalFloat) * time.Minute
	// 	} else {
	// 		log.Printf("Invalid type for 'processTimeInterval': %v\n", interval)
	// 	}
	// }

	// if idleFetch, exists := responseMap["idleThreshold"]; exists {
	// 	if idleFloat, ok := idleFetch.(float64); ok {
	// 		idleThreshold = time.Duration(idleFloat) * time.Minute
	// 	} else {
	// 		log.Printf("Invalid type for 'idleThreshold': %v\n", idleFetch)
	// 	}
	// }

	return responseMap, nil
}

func AddUser(machineID string, email string, employeeId string) (map[string]interface{}, error) {
	apiURL := ADDUSER_URL
	if apiURL == "" {
		return nil, fmt.Errorf("API_URL environment variable not set")
	}
	if employeeId == "" || email == "" || machineID == "" {
		return nil, fmt.Errorf("employeeId, email, machineID cannot be empty")
	}
	requestBody := map[string]interface{}{
		"machineID":  machineID,
		"email":      email,
		"employeeId": employeeId,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling data to JSON: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("error unmarshalling response body", err)

		return nil, fmt.Errorf("error creating POST request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error unmarshalling response body", err)

		return nil, fmt.Errorf("error sending POST request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error unmarshalling response body", err)

	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		fmt.Println("error unmarshalling response body", err, resp.StatusCode)
		return nil, fmt.Errorf("received non-OK response: %s", body)
	}

	var responseMap map[string]interface{}
	err = json.Unmarshal(body, &responseMap)
	if err != nil {
		fmt.Println("error unmarshalling response body")
	}

	return responseMap, nil
}
func sendProcess(data interface{}) error {
	apiURL := PROCESS_URL
	if apiURL == "" {
		return fmt.Errorf("API_URL environment variable not set")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshalling data to JSON: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating POST request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending POST request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response: %s", body)
	}

	return nil
}

func isNetworkAvailable() bool {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("https://www.google.com")
	if err != nil {
		fmt.Printf("Network check failed: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true
	}

	return false
}
