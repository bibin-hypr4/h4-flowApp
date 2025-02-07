package main

import (
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/go-vgo/robotgo"
	"github.com/kbinani/screenshot"
	"github.com/shirou/gopsutil/process"
)

var checkedIn, idleStatus bool
var checkinTime, checkoutTime, sessionStart, sessionEnd, SleepTime time.Time
var workingTime, sessionTime time.Duration
var checkInstatus string
var mu sync.Mutex
var desk desktop.App
var menItems *fyne.Menu

var mItemCheckin, mUser, mSession, mWorkTime, mIdle, mQuit, mAbout *fyne.MenuItem
var machineID string
var userEmail string
var processTimeInverval time.Duration // in minutes
var idleThreshold time.Duration
var dailyIdleTime time.Duration
var processTimes = make(map[int]time.Duration) // Map to store time spent per process
var processLastSeen = make(map[int]time.Time)
var USER_IP string
var IdleIcon fyne.Resource
var FyneAPP fyne.App

var (
	IDLE_URL       = "https://h4api.muxly.app/api/attendance/v4/record/idle"
	ATTENDANCE_URL = "https://h4api.muxly.app/api/attendance/v4/record/add"
	PROCESS_URL    = "https://h4api.muxly.app/api/attendance/v4/record/process"
	GETUSER_URL    = "https://h4api.muxly.app/api/attendance/v4/user/get"
	ADDUSER_URL    = "https://h4api.muxly.app/api/attendance/v4/user/register"
	CONFIG_URL     = "https://h4api.muxly.app/api/attendance/v4/config"
	UPLOAD_URL     = "https://h4api.muxly.app/api/attendance/v4/upload"
	VERSION        = "1.3.4"
)

func Init() {
	go monitorSleepWindows()
	logFile = getAppDataDir() + "/attendance.log"
	USER_IP, _ = getPublicIP()
	id, err := GetMachineID()
	if err != nil {
		log.Fatalf("Error getting machine ID: %v", err)
	}
	machineID = id
	//fetch users details
	userDetails, err := getUserDetails(machineID)
	if err != nil {
		log.Println("err", err)
		email, _, err := getUserInput()
		if err != nil {
			log.Println("err", err)
			log.Fatalf("Error getting user input: %v", err)
		}

		userEmail = email
	} else {
		userEmail = userDetails["email"].(string)
	}
}

func main() {

	fmt.Println("started")
	defer func() {
		if r := recover(); r != nil {
			log.Print("crashed")
			handleCrash(r)
		} else {
			log.Print("crashed")
			handleCrash("closing app")
		}
	}()

	Init()

	// Handle system signals for graceful shutdown
	go handleShutdownWindows()
	res, err := FetchConfigDetails(CONFIG_URL)
	if err != nil {
		log.Fatalf("failed to fetch config details")
	}
	app := app.NewWithID("h4-Flow App")
	FyneAPP = app
	version, exist := res["version"].(string)
	if !exist {
		version = VERSION
	}

	if version != VERSION {
		showAppError(" Please update the App to latest version", app)
	}
	processTimeInverval = 15 * time.Minute
	idleThreshold = 15 * time.Minute
	initializeApp(app)

}
func updateTrayMenu(desk desktop.App) {
	separator := fyne.NewMenuItemSeparator()
	menItems = fyne.NewMenu("H4 - Flow", mUser, mAbout, mItemCheckin, separator, mSession, mIdle, separator, mQuit)
	desk.SetSystemTrayMenu(menItems)
}

func initializeApp(a fyne.App) {
	// Load the idle icon for the tray
	idleIcon, _ := fyne.LoadResourceFromPath("./idle.ico")
	activeIcon, _ := fyne.LoadResourceFromPath("./active.ico")
	IdleIcon = idleIcon
	// idleIcon := fyne.NewStaticResource("idle.ico", idleImageData)
	// activeIcon := fyne.NewStaticResource("active.ico", activeImageData)

	// Set the initial status text for the check-in
	userName := currentUsername()

	// Create the tray menu items
	var ok bool
	if desk, ok = a.(desktop.App); ok {
		mItemCheckin = fyne.NewMenuItem("Check In", func() {
			if !checkedIn && !isNetworkAvailable() {
				mItemCheckin.Label = "Check In - network error"
				return
			}

			checkActivity()
			if checkedIn {

				desk.SetSystemTrayIcon(activeIcon)

			} else {

				mItemCheckin.Label = "Check In"

				desk.SetSystemTrayIcon(idleIcon)
			}

			menItems.Refresh()

		})

		mUser = fyne.NewMenuItem("Hello "+userName, func() {
		})
		mAbout = fyne.NewMenuItem("About", func() {
			// This will be triggered when the "About" menu is clicked
		})
		subMenu := fyne.NewMenu(
			"Details",
			fyne.NewMenuItem("App Name: h4-FlowApp", nil),
			fyne.NewMenuItem(fmt.Sprintf("Version: %s", VERSION), nil),
		)
		mAbout.ChildMenu = subMenu
		// mWorkTime = fyne.NewMenuItem("Work Time", func() {
		// })
		mSession = fyne.NewMenuItem("Session", func() {
		})
		mIdle = fyne.NewMenuItem("Idle", func() {
			println("Idle clicked")
		})

		mQuit = fyne.NewMenuItem("Quit", func() {
			a.Quit()
		})
		updateTrayMenu(desk)

		// Create the tray menu and assign it to the tray
		// m := fyne.NewMenu("H4 - Flow", mItemCheckin, mWorkTime, mIdle, mQuit)

		// Set the system tray menu and icon
		desk.SetSystemTrayIcon(idleIcon)
		go func() {

			for range time.Tick(time.Second * 1) {
				if !checkedIn {
					continue
				}
				now := time.Now()

				// Check if the current time is 11:59 PM
				if now.Hour() == 23 && now.Minute() == 59 {
					checkActivity()
					checkActivity()
				}
				// sec := (int32)(workingTime.Seconds())
				// ws := fmt.Sprintf("Work - %02d:%02d:%02d", (sec / 3600), (sec%3600)/60, (sec % 60))
				// mWorkTime.Label = ws
				isec := (int32)(dailyIdleTime.Seconds())
				idles := fmt.Sprintf("Idle - %02d:%02d:%02d", (isec / 3600), (isec%3600)/60, (isec % 60))
				mIdle.Label = idles

				if !idleStatus {
					sec := (int32)(sessionTime.Seconds())
					ss := fmt.Sprintf("Session - %02d:%02d:%02d", (sec / 3600), (sec%3600)/60, (sec % 60))
					sessionTime += time.Second

					mSession.Label = ss
				}

				menItems.Refresh()
				// updateTrayMenu(desk)

			}
		}()
	}

	go func() {
		for range time.Tick(processTimeInverval) {
			if checkedIn {
				processList()
			}
			processLogs()

		}
	}()
	a.Run()
}

func checkActivity() {
	if !checkedIn {

		if !isSameDay() {
			dailyIdleTime = 0
			workingTime = 0
			if !isLatestApp() {
				showAppError("This version of app is outdated", FyneAPP)
			}
		}
		go func() {
			getIdleTime()
		}()
		checkedIn = true
		mItemCheckin.Label = "Checkout"
		mItemCheckin.Checked = true
		updateCheckinTime()
		recordAttendance("attendance", "checked_in", machineID, checkinTime, checkoutTime, workingTime, dailyIdleTime)
		sessionStart = time.Now()
		recordAttendance("session", "checked_in", machineID, sessionStart, time.Now(), sessionTime, dailyIdleTime)

		sessionEnd = time.Time{}

	} else {
		checkedIn = false
		mItemCheckin.Label = "Check In"
		mItemCheckin.Checked = false
		updateCheckoutTime()
		recordAttendance("attendance", "checked_out", machineID, checkinTime, checkoutTime, workingTime, dailyIdleTime)
		recordAttendance("session", "checked_out", machineID, sessionStart, time.Now(), sessionTime, dailyIdleTime)
		sessionStart = time.Time{}
		sessionEnd = time.Time{}
	}
}

func getIdleTime() {
	lastActive := time.Now()
	lastx, lasty := robotgo.Location()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var idleStartTime time.Time

	for {
		if !checkedIn {
			fmt.Println("CheckedIn is false, stopping tracking.")
			return
		}

		// Track mouse events
		x, y := robotgo.Location()
		if x != lastx || y != lasty {
			// Mouse moved, reset idle time
			if !idleStartTime.IsZero() {
				idleStartTime = time.Time{}
				if idleStatus && time.Since(idleStartTime) >= idleThreshold {
					sessionStart = time.Now()
					recordAttendance("session", "checked_in", machineID, sessionStart, time.Now(), sessionTime, dailyIdleTime)

					idleStatus = false
				}
			}

			lastActive = time.Now()
			lastx, lasty = x, y
		}

		// Check if idle time should be recorded
		// Track idle periods
		if time.Since(lastActive) > idleThreshold && idleStartTime.IsZero() {
			sessionEnd = time.Now()
			idleStatus = true
			// fmt.Println("idle started", sessionStart, sessionEnd)
			// // recordSession()
			recordAttendance("session", "checked_out", machineID, sessionStart, sessionEnd, sessionTime, dailyIdleTime)
			sessionStart = time.Time{}
			sessionEnd = time.Time{}
			idleStartTime = lastActive
		} else if time.Since(lastActive) <= idleThreshold && !idleStartTime.IsZero() {
			if sessionStart.IsZero() {
				sessionStart = time.Now()
				recordAttendance("session", "checked_in", machineID, sessionStart, time.Time{}, sessionTime, dailyIdleTime)

			}
			idleEndTime := time.Now()
			dailyIdleTime += time.Since(idleStartTime) - idleThreshold
			// recordIdleTime(idleStartTime, idleEndTime)
			sessionTime = 0
			// updateCheckinTime()
			idleStartTime = time.Time{}
			fmt.Println("Mouse moved, idle ended at: ", idleEndTime, dailyIdleTime)

		}

	}
}
func processList() interface{} {
	// List only current user processes
	processes, err := process.Processes()
	if err != nil {
		log.Println("Error retrieving processes:", err)
		return nil
	}

	data := []ProcessInfo{}
	now := time.Now() // Current timestamp

	for _, proc := range processes {
		// Ensure we're retrieving non-nil or empty values
		name, err := proc.Name()
		if err != nil || name == "" {
			continue // Skip if name is invalid or empty
		}

		// status, err := proc.Status()
		// if err != nil {
		// 	status = "unknown" // Default to "unknown" if unable to get status
		// }

		user, err := proc.Username()
		if err != nil || user != currentUsername() {
			continue // Skip if the process is not running by the current user
		}

		cpu, err := proc.CPUPercent()
		if err != nil || cpu <= 1 {
			continue // Skip if CPU usage is <= 1% or unable to get the CPU percentage
		}

		running, err := proc.IsRunning()
		if err != nil || !running {
			continue // Skip if the process is not running
		}

		pid := proc.Pid

		// Calculate time spent for this process
		if lastSeen, exists := processLastSeen[int(pid)]; exists {
			duration := now.Sub(lastSeen)      // Time since the last check
			processTimes[int(pid)] += duration // Add to cumulative time
		}
		processLastSeen[int(pid)] = now // Update last seen time

		row := ProcessInfo{
			PID:       fmt.Sprintf("%d", pid),
			Name:      name,
			User:      user,
			Cpu:       fmt.Sprintf("%2.2f%%", cpu),
			Email:     userEmail,
			Type:      "process",
			Date:      now.Format("2006-01-02"),
			Timestamp: now.String(),
			TimeSpent: processTimes[int(pid)].String(),
		}
		data = append(data, row)
	}

	// Clean up terminated processes
	for pid := range processLastSeen {
		found := false
		for _, proc := range processes {
			if proc.Pid == int32(pid) {
				found = true
				break
			}
		}
		if !found {
			delete(processTimes, pid)
			delete(processLastSeen, pid)
		}
	}
	go sendProcess(data)
	return data
}
func CaptureScreenshots() ([]string, error) {
	var savedFiles []string
	n := screenshot.NumActiveDisplays()
	if n < 1 {
		return nil, fmt.Errorf("no active displays found")
	}

	// Ensure the "screenshots" folder exists
	dirPath := "screenshots"
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create screenshots directory: %v", err)
	}

	// Loop through each display and capture a screenshot
	for i := 0; i < n; i++ {
		// Capture the display screenshot
		img, err := screenshot.CaptureDisplay(i)
		if err != nil {
			log.Printf("Failed to capture display %d: %v", i, err)
			continue
		}

		timestamp := time.Now().Format("20060102_150405")
		fileName := fmt.Sprintf("screenshot_display_%d_%s.png", i, timestamp)
		filePath := filepath.Join(dirPath, fileName)

		// Save the screenshot as a PNG file
		file, err := os.Create(filePath)
		if err != nil {
			log.Printf("Failed to save screenshot for display %d: %v", i, err)
			continue
		}
		defer file.Close()

		if err := png.Encode(file, img); err != nil {
			log.Printf("Failed to encode image for display %d: %v", i, err)
			continue
		}

		// Add the file path to the list of saved files
		savedFiles = append(savedFiles, filePath)
	}

	// Return the list of saved file paths
	if len(savedFiles) == 0 {
		return nil, fmt.Errorf("no screenshots were saved")
	}

	return savedFiles, nil
}
