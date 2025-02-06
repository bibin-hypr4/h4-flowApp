package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/MarinX/keylogger"
	"github.com/denisbrodbeck/machineid"
	"github.com/godbus/dbus/v5"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
)

func getUserInput() (string, string, error) {
	a := app.New()
	w := a.NewWindow("User Information")

	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("Enter your email")

	employeeIDEntry := widget.NewEntry()
	employeeIDEntry.SetPlaceHolder("Enter your employee ID (capital E followed by 4 digits)")

	var email, employeeID string
	var err error

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Email", Widget: emailEntry},
			{Text: "Employee ID", Widget: employeeIDEntry},
		},
		OnSubmit: func() {
			email = emailEntry.Text
			employeeID = employeeIDEntry.Text

			// Validate email
			email = strings.ToLower(email)
			emailRegex := regexp.MustCompile(`^[a-z0-9._%+-]+@hypr4\.io$`)
			if !emailRegex.MatchString(email) {
				dialog.ShowError(fmt.Errorf("invalid email format. It must end with @hypr4.io and be lowercase"), w)
				return
			}

			// Validate employee ID
			employeeIDRegex := regexp.MustCompile(`^[A-Z]\d{4}$`)
			if !employeeIDRegex.MatchString(employeeID) {
				dialog.ShowError(fmt.Errorf("invalid employee ID format. It must start with Capital letter followed by 4 digits"), w)
				return
			}

			w.Close()
		},
	}

	w.SetContent(container.NewVBox(
		form,
	))
	w.Resize(fyne.NewSize(500, 200)) // Adjust the width and height as needed
	w.CenterOnScreen()
	w.ShowAndRun()

	return email, employeeID, err
}

// If today is not holiday, user is not on leave and first time checkin, mark attendance
func markAttendance() {
	// TODO - get holiday and leave from API
	holiday := false
	userOnLeave := false

	if !holiday && !userOnLeave && !checkedIn {
		// mAttendance.Check()
	}
}

// workingTime starts at 0, each checkin to checkout will be added for the day
func updateCheckinTime() {
	markAttendance()
	checkinTime = time.Now()
	workingTime += sessionTime
	sessionTime = workingTime
	sessionTime = 0
}

func updateCheckoutTime() {
	checkoutTime = time.Now()
	sessionTime = checkoutTime.Sub(checkinTime)
	workingTime += sessionTime
	sessionTime = 0
}

// Capture Screen
// func captureScreen() {
// 	n := screenshot.NumActiveDisplays()

// 	for i := 0; i < n; i++ {
// 		bounds := screenshot.GetDisplayBounds(i)
// 		img, err := screenshot.CaptureRect(bounds)
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 		fileName := fmt.Sprintf("./capture/%s-%d-%d-%dx%d.png", currentUsername(),
// 			i, time.Now().Unix(), bounds.Dx(), bounds.Dy())
// 		file, _ := os.Create(fileName)
// 		defer file.Close()
// 		png.Encode(file, img)
// 	}
// }

// END ACTIVITY

func onExit() {
	// Cleaning stuff here.
	fmt.Printf("exiting...")
}

func getIcon(s string) []byte {
	b, err := ioutil.ReadFile(s)
	if err != nil {
		fmt.Print(err)
	}
	return b
}

// System Utility Functions
// START - System Information Utility functions
// Host Info
// CPU Usage
// Network Info
// Process List

func currentUsername() string {
	cu, _ := user.Current()
	return cu.Username
}

func hostInfo() string {
	hInfo, _ := host.Info()
	return fmt.Sprintf("Host %s", hInfo.Hostname)
}

func cpuInfo() string {
	cpuUsage, _ := cpu.Percent(time.Second, false)
	return fmt.Sprintf("CPU %f", cpuUsage)
}
func GetHostMAC() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatal("Error fetching interfaces:", err)
	}

	for _, iface := range interfaces {
		if iface.HardwareAddr != nil {
			return iface.HardwareAddr.String()
		}
	}
	return ""
}
func networkInfo() string {
	addrs, err := net.InterfaceAddrs()
	var localIP, currentNetworkHardwareName string
	if err != nil {
		return "ERROR"
	}
	for _, addr := range addrs {
		ipAddr, ok := addr.(*net.IPNet)
		if !ok || ipAddr.IP.IsLoopback() || !ipAddr.IP.IsGlobalUnicast() {
			continue
		}
		localIP = ipAddr.IP.String()
	}

	interfaces, _ := net.Interfaces()
	for _, interf := range interfaces {
		if addrs, err := interf.Addrs(); err == nil {
			for _, addr := range addrs {
				if strings.Contains(addr.String(), localIP) {
					currentNetworkHardwareName = interf.Name
				}
			}
		}
	}
	netInterface, err := net.InterfaceByName(currentNetworkHardwareName)
	macAddr := netInterface.HardwareAddr

	info := fmt.Sprintf("Local IP : %s\nMAC: %s", localIP, macAddr)
	return info
}

type LASTINPUTINFO struct {
	CbSize uint32
	DwTime uint32
}

// Function to handle key press events and reset idle time
func handleKeyPress(event keylogger.InputEvent, idleStartTime *time.Time, lastActive *time.Time) {
	// Only handle key press events (EvKey type and value 1 for pressed)
	if event.Type == keylogger.EvKey && event.Value == 1 {
		// Reset idle start time if idle period ended
		if !(*idleStartTime).IsZero() {
			idleEndTime := time.Now()
			idleDuration := idleEndTime.Sub(*idleStartTime)
			fmt.Printf("Idle started at: %v, ended at: %v, duration: %v\n", *idleStartTime, idleEndTime, idleDuration)
			*idleStartTime = time.Time{}
		}
		// Record activity
		*lastActive = time.Now()
		// fmt.Printf("Key pressed: %v\n", event.KeyString)
	}
}

// END - System Information Utility functions
func loadIcon(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to load icon: %v", err)
	}
	return data
}

func isSameDay() bool {
	if checkinTime.IsZero() {
		checkinTime = time.Now()
	}
	return checkinTime.Day() == time.Now().Day()
}

// func recordSession() {
// 	tempTime := time.Time{}
// 	if tempTime == sessionStart {
// 		return
// 	}

//		sessionEnd = time.Now()
//		fmt.Println("session start,end", sessionStart, sessionEnd)
//		recordAttendance("session", "check_out", machineID, sessionStart, sessionEnd, sessionTime, dailyIdleTime)
//		sessionStart = time.Time{}
//	}
func GetMachineID() (string, error) {
	var id string
	var err error

	switch runtime.GOOS {
	case "linux":

		id, err = machineid.ID()
		if err != nil {
			return "", fmt.Errorf("error getting machine ID on Linux: %v", err)
		}
	case "windows":
		// For Windows, use the machineid package
		id, err = machineid.ID()
		if err != nil {
			return "", fmt.Errorf("error getting machine ID on Windows: %v", err)
		}
	case "darwin":
		// For macOS, use the machineid package
		id, err = machineid.ID()
		if err != nil {
			return "", fmt.Errorf("error getting machine ID on macOS: %v", err)
		}
	default:
		return "", fmt.Errorf("unsupported OS: %v", runtime.GOOS)
	}

	return id, nil
}
func handleSignals() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		handleCrash(sig)
		fmt.Println("Received signal:", sig)
		os.Exit(0)
	}()
}

func showAppError(message string, app fyne.App) {
	window := app.NewWindow("H4 Flow App")
	window.Resize(fyne.NewSize(400, 300)) // Set a default size for the window
	window.CenterOnScreen()
	window.Show() // Ensure the window is shown

	// Show version mismatch dialog
	dialog := dialog.NewInformation(
		"Version Mismatch",
		message,

		window,
	)
	dialog.SetOnClosed(func() {
		window.Close() // Close the window when the dialog is closed
		app.Quit()
		log.Fatalf("")
		// Quit the application
	})
	// Ensure dialog blocks app execution until user responds
	dialog.Show()

	// Wait for dialog to be closed, don't proceed further until user clicks OK or Cancel
	// app.Run()
	// app.Quit()
	// return

}
func getPublicIP() (string, error) {
	resp, err := http.Get("https://api64.ipify.org?format=text")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(ip), nil
}
func UploadAllScreenshots() ([]string, error) {
	// Get the screenshots directory path
	dirPath := "screenshots"

	// Read all files in the directory
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read screenshots directory: %v", err)
	}

	var uploadedFiles []string
	for _, file := range files {
		// Only upload .png files
		if file.IsDir() || filepath.Ext(file.Name()) != ".png" {
			continue
		}

		// Get the full file path
		filePath := filepath.Join(dirPath, file.Name())

		// Upload the screenshot
		err := UploadScreenshot(filePath, UPLOAD_URL)
		if err != nil {
			log.Printf("Failed to upload file %s: %v", filePath, err)
			continue
		}

		// Add the file to the uploaded list
		uploadedFiles = append(uploadedFiles, filePath)

		// Delete the file after a successful upload
		err = os.Remove(filePath)
		if err != nil {
			log.Printf("Failed to delete file %s: %v", filePath, err)
		} else {
			log.Printf("Successfully uploaded and deleted file: %s", filePath)
		}
	}

	// Return the list of uploaded files
	if len(uploadedFiles) == 0 {
		return nil, fmt.Errorf("no screenshots were uploaded")
	}

	return uploadedFiles, nil
}
func UploadScreenshot(filePath, uploadURL string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Create file part
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file for %s: %v", filePath, err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file content for %s: %v", filePath, err)
	}

	// Add additional form fields
	_ = writer.WriteField("email", userEmail)
	_ = writer.WriteField("ip", USER_IP)
	_ = writer.WriteField("uploadtime", time.Now().GoString()) // Should be in "YYYY-MM-DD HH:MM:SS" format
	_ = writer.WriteField("machineid", machineID)

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close multipart writer: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", uploadURL, &requestBody)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("upload failed with status %s", resp.Status)
	}

	return nil
}

func isLatestApp() bool {
	res, err := FetchConfigDetails(CONFIG_URL)
	if err != nil {
		return false
	}
	version, exist := res["version"].(string)
	if !exist {
		version = VERSION
	}
	if version != VERSION {
		return false
	}
	return true
}
func ShowError(version string, app fyne.App) {
	window := app.NewWindow("H4 Flow App")
	window.Resize(fyne.NewSize(400, 300)) // Set a default size for the window
	window.CenterOnScreen()
	window.Show() // Ensure the window is shown

	// Show version mismatch dialog
	dialog := dialog.NewInformation(
		"Version Mismatch",
		fmt.Sprintf("Invalid version detected. Current version: %s. Please update the app.", version),

		window,
	)
	dialog.SetOnClosed(func() {
		window.Close() // Close the window when the dialog is closed
		app.Quit()     // Quit the application
	})

	// Ensure dialog blocks app execution until user responds
	dialog.Show()

	// Wait for dialog to be closed, don't proceed further until user clicks OK or Cancel
	app.Run()
	app.Quit()
	return

}

// getAppDataDir returns a secure directory for storing application data
func getAppDataDir() string {
	var baseDir string

	if runtime.GOOS == "windows" {
		// Windows: Use %APPDATA%\yourapp\data
		appData := os.Getenv("APPDATA")
		if appData != "" {
			baseDir = filepath.Join(appData, "flow-app", ".data")
		} else {
			// Fallback to local directory if APPDATA is not set
			baseDir = filepath.Join(".", "data")
		}
	} else {
		// Linux/macOS: Use ~/.yourapp/data
		if homeDir, err := os.UserHomeDir(); err == nil {
			baseDir = filepath.Join(homeDir, "flow-app", ".data")
		} else {
			// Fallback to local directory if home dir cannot be found
			baseDir = filepath.Join(".", "data")
		}
	}

	// Ensure the directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	return baseDir
}

func monitorSleepLinux() {
	deleteAttendanceRecords()
	conn, err := dbus.SystemBus()
	if err != nil {
		fmt.Println("Failed to connect to DBus:", err)
		return
	}

	// Listen for system sleep signals
	rule := "type='signal',interface='org.freedesktop.login1.Manager',member='PrepareForSleep'"
	call := conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule)
	if call.Err != nil {
		fmt.Println("Failed to add DBus match:", call.Err)
		return
	}

	signals := make(chan *dbus.Signal, 10)
	conn.Signal(signals)

	for sig := range signals {
		if !checkedIn {
			fmt.Println("not checked in")
			continue
		}
		if len(sig.Body) > 0 {
			if !checkedIn {
				fmt.Println("not checked in")
				continue
			}
			if sleeping, ok := sig.Body[0].(bool); ok && sleeping {
				fmt.Println("System is going to sleep!")

				recordDetails := AttendanceRecord{
					Type:         "session",
					Status:       "checked_out",
					Email:        userEmail,
					MachineID:    machineID,
					RecordTime:   time.Now().Format("2006-01-02T15:04:05.999999999-07:00"),
					CheckinTime:  sessionStart.Format("2006-01-02T15:04:05.999999999-07:00"),
					CheckoutTime: time.Now().Format("2006-01-02T15:04:05.999999999-07:00"),
					WorkingTime:  workingTime.Hours(),
					IdleTime:     dailyIdleTime.Hours(),
					WorktimeMin:  workingTime.Minutes(),
					Date:         time.Now().Format("2006-01-02"),
					IP:           USER_IP,
				}
				writeAttendanceRecord(recordDetails)
			} else {
				// System is waking up
				checkinTime = time.Now()
				sessionStart = time.Now()
				sessionEnd = time.Time{}
				sessionTime = 0
				checkoutTime = time.Time{}
				logRecords, _ := readAttendanceRecords()
				err := sendPostRequest(logRecords, "attendance")
				fmt.Println("err", err)

				if err != nil {
					go deleteAttendanceRecords()
				}

				fmt.Println("System is waking up!", time.Now())
			}
		}
	}
}
