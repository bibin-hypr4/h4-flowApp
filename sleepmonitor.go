package main

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/gonutz/w32"
)

const (
	WM_POWERBROADCAST    = 0x218
	PBT_APMSUSPEND       = 0x4 // System is suspending
	PBT_APMRESUMESUSPEND = 0x7 // System has resumed
)

func monitorSleepWindows() {

	// Convert string to UTF-16 pointer
	wndClassName := syscall.StringToUTF16Ptr("SleepMonitorWindow")

	// Get module handle
	hInstance, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetModuleHandleW").Call(0)

	// Register the window class
	wndClass := w32.WNDCLASSEX{
		Size:      uint32(unsafe.Sizeof(w32.WNDCLASSEX{})),
		WndProc:   syscall.NewCallback(windowProc),
		Instance:  w32.HINSTANCE(hInstance),
		ClassName: wndClassName,
	}

	if w32.RegisterClassEx(&wndClass) == 0 {
		fmt.Println("Failed to register window class")
		return
	}

	// Create the hidden window to listen for events
	hwnd := w32.CreateWindowEx(0, wndClassName, syscall.StringToUTF16Ptr("Sleep Monitor"), 0, 0, 0, 0, 0, 0, 0, w32.HINSTANCE(hInstance), nil)
	if hwnd == 0 {
		fmt.Println("Failed to create window")
		return
	}

	// Message loop to capture power events
	var msg w32.MSG
	for w32.GetMessage(&msg, hwnd, 0, 0) > 0 {
		w32.TranslateMessage(&msg)
		w32.DispatchMessage(&msg)
	}

}

func windowProc(hwnd w32.HWND, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case WM_POWERBROADCAST:
		if wparam == PBT_APMSUSPEND {
			fmt.Println("System is going to sleep!", time.Now())

			if !checkedIn {
				fmt.Println("Not checked in")
				return 1
			}

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
		} else if wparam == PBT_APMRESUMESUSPEND {
			fmt.Println("System is waking up!", time.Now())

			checkinTime = time.Now()
			sessionStart = time.Now()
			sessionEnd = time.Time{}
			sessionTime = 0
			checkoutTime = time.Time{}

			logRecords, _ := readAttendanceRecords()
			if len(logRecords) > 0 {
				err := sendPostRequest(logRecords, "attendance")
				fmt.Println("err", err)

				if err != nil {
					go deleteAttendanceRecords()
				}
			}
		}
		return 1
	default:
		return w32.DefWindowProc(hwnd, msg, wparam, lparam)
	}
}
