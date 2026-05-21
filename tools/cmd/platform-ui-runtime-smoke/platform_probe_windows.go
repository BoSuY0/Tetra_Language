//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	wsOverlappedWindow = 0x00CF0000
	wsChild            = 0x40000000
	wsVisible          = 0x10000000
	esAutoHScroll      = 0x00000080
	lbsNotify          = 0x00000001
	swShow             = 5
	cwUseDefault       = 0x80000000
	wmSetText          = 0x000C
	wmTimer            = 0x0113
	bmClick            = 0x00F5
	lbAddString        = 0x0180
	lbSetCurSel        = 0x0186
	pmRemove           = 0x0001
	rdwInvalidate      = 0x0001
	rdwAllChildren     = 0x0080
	rdwUpdateNow       = 0x0100
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type point struct {
	X int32
	Y int32
}

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

func runPlatformWindowProbe(target string) (platformWindowProbeResult, error) {
	if target != "windows-x64" {
		return platformWindowProbeResult{}, fmt.Errorf("windows platform probe cannot run target %s", target)
	}
	user32 := syscall.NewLazyDLL("user32.dll")
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getModuleHandle := kernel32.NewProc("GetModuleHandleW")
	registerClassEx := user32.NewProc("RegisterClassExW")
	createWindowEx := user32.NewProc("CreateWindowExW")
	showWindow := user32.NewProc("ShowWindow")
	updateWindow := user32.NewProc("UpdateWindow")
	destroyWindow := user32.NewProc("DestroyWindow")
	unregisterClass := user32.NewProc("UnregisterClassW")
	defWindowProc := user32.NewProc("DefWindowProcW")
	sendMessage := user32.NewProc("SendMessageW")
	setFocus := user32.NewProc("SetFocus")
	setTimer := user32.NewProc("SetTimer")
	killTimer := user32.NewProc("KillTimer")
	postMessage := user32.NewProc("PostMessageW")
	peekMessage := user32.NewProc("PeekMessageW")
	translateMessage := user32.NewProc("TranslateMessage")
	dispatchMessage := user32.NewProc("DispatchMessageW")
	redrawWindow := user32.NewProc("RedrawWindow")

	instance, _, err := getModuleHandle.Call(0)
	if instance == 0 {
		return platformWindowProbeResult{}, fmt.Errorf("GetModuleHandleW failed: %v", err)
	}
	className, _ := syscall.UTF16PtrFromString("TetraPlatformUIRuntimeSmoke")
	title, _ := syscall.UTF16PtrFromString("Tetra UI Runtime Smoke")
	wndProc := syscall.NewCallback(func(hwnd uintptr, msg uint32, wparam uintptr, lparam uintptr) uintptr {
		ret, _, _ := defWindowProc.Call(hwnd, uintptr(msg), wparam, lparam)
		return ret
	})
	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   wndProc,
		Instance:  instance,
		ClassName: className,
	}
	if atom, _, err := registerClassEx.Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
		return platformWindowProbeResult{}, fmt.Errorf("RegisterClassExW failed: %v", err)
	}
	hwnd, _, err := createWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		wsOverlappedWindow,
		cwUseDefault,
		cwUseDefault,
		320,
		200,
		0,
		0,
		instance,
		0,
	)
	if hwnd == 0 {
		_, _, _ = unregisterClass.Call(uintptr(unsafe.Pointer(className)), instance)
		return platformWindowProbeResult{}, fmt.Errorf("CreateWindowExW failed: %v", err)
	}
	showWindow.Call(hwnd, swShow)
	updateWindow.Call(hwnd)
	label, err := createWin32Control(createWindowEx, "STATIC", "Editing", 16, 16, 280, 24, hwnd, 101, instance, 0)
	if err != nil {
		destroyWindow.Call(hwnd)
		unregisterClass.Call(uintptr(unsafe.Pointer(className)), instance)
		return platformWindowProbeResult{}, err
	}
	edit, err := createWin32Control(createWindowEx, "EDIT", "tetra", 16, 48, 280, 24, hwnd, 102, instance, esAutoHScroll)
	if err != nil {
		destroyWindow.Call(hwnd)
		unregisterClass.Call(uintptr(unsafe.Pointer(className)), instance)
		return platformWindowProbeResult{}, err
	}
	list, err := createWin32Control(createWindowEx, "LISTBOX", "", 16, 80, 280, 72, hwnd, 103, instance, lbsNotify)
	if err != nil {
		destroyWindow.Call(hwnd)
		unregisterClass.Call(uintptr(unsafe.Pointer(className)), instance)
		return platformWindowProbeResult{}, err
	}
	button, err := createWin32Control(createWindowEx, "BUTTON", "Save", 16, 160, 120, 28, hwnd, 104, instance, 0)
	if err != nil {
		destroyWindow.Call(hwnd)
		unregisterClass.Call(uintptr(unsafe.Pointer(className)), instance)
		return platformWindowProbeResult{}, err
	}
	if err := exerciseWin32UIRuntime(hwnd, label, edit, list, button, sendMessage, setFocus, setTimer, killTimer, postMessage, peekMessage, translateMessage, dispatchMessage, redrawWindow); err != nil {
		destroyWindow.Call(hwnd)
		unregisterClass.Call(uintptr(unsafe.Pointer(className)), instance)
		return platformWindowProbeResult{}, err
	}
	destroyWindow.Call(hwnd)
	unregisterClass.Call(uintptr(unsafe.Pointer(className)), instance)
	return platformWindowProbeResult{
		API:     "win32-user32",
		Markers: []string{"platform-window-create:win32-user32", "platform-widget-tree:ok", "platform-event-dispatch:ok", "platform-timer:ok", "platform-redraw:ok", "platform-window-close:win32-user32"},
	}, nil
}

func createWin32Control(createWindowEx *syscall.LazyProc, className string, title string, x int32, y int32, width int32, height int32, parent uintptr, id uintptr, instance uintptr, style uintptr) (uintptr, error) {
	classPtr, _ := syscall.UTF16PtrFromString(className)
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	hwnd, _, err := createWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(classPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		wsChild|wsVisible|style,
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		parent,
		id,
		instance,
		0,
	)
	if hwnd == 0 {
		return 0, fmt.Errorf("CreateWindowExW %s failed: %v", className, err)
	}
	return hwnd, nil
}

func exerciseWin32UIRuntime(hwnd uintptr, label uintptr, edit uintptr, list uintptr, button uintptr, sendMessage *syscall.LazyProc, setFocus *syscall.LazyProc, setTimer *syscall.LazyProc, killTimer *syscall.LazyProc, postMessage *syscall.LazyProc, peekMessage *syscall.LazyProc, translateMessage *syscall.LazyProc, dispatchMessage *syscall.LazyProc, redrawWindow *syscall.LazyProc) error {
	item1, _ := syscall.UTF16PtrFromString("item-1")
	item2, _ := syscall.UTF16PtrFromString("item-2")
	input, _ := syscall.UTF16PtrFromString("tetra-ui")
	saved, _ := syscall.UTF16PtrFromString("Saved")
	timerSaved, _ := syscall.UTF16PtrFromString("Saved after timer")
	setFocus.Call(edit)
	if ok, _, err := sendMessage.Call(edit, wmSetText, 0, uintptr(unsafe.Pointer(input))); ok == 0 {
		return fmt.Errorf("SendMessageW WM_SETTEXT failed: %v", err)
	}
	if idx, _, err := sendMessage.Call(list, lbAddString, 0, uintptr(unsafe.Pointer(item1))); idx == ^uintptr(0) {
		return fmt.Errorf("SendMessageW LB_ADDSTRING item-1 failed: %v", err)
	}
	if idx, _, err := sendMessage.Call(list, lbAddString, 0, uintptr(unsafe.Pointer(item2))); idx == ^uintptr(0) {
		return fmt.Errorf("SendMessageW LB_ADDSTRING item-2 failed: %v", err)
	}
	if idx, _, err := sendMessage.Call(list, lbSetCurSel, 1, 0); idx == ^uintptr(0) {
		return fmt.Errorf("SendMessageW LB_SETCURSEL failed: %v", err)
	}
	sendMessage.Call(button, bmClick, 0, 0)
	if ok, _, err := sendMessage.Call(label, wmSetText, 0, uintptr(unsafe.Pointer(saved))); ok == 0 {
		return fmt.Errorf("SendMessageW label save state failed: %v", err)
	}
	timer, _, err := setTimer.Call(hwnd, 1, 10, 0)
	if timer == 0 {
		return fmt.Errorf("SetTimer failed: %v", err)
	}
	if ok, _, err := postMessage.Call(hwnd, wmTimer, timer, 0); ok == 0 {
		killTimer.Call(hwnd, timer)
		return fmt.Errorf("PostMessageW WM_TIMER failed: %v", err)
	}
	var m msg
	if got, _, err := peekMessage.Call(uintptr(unsafe.Pointer(&m)), hwnd, 0, 0, pmRemove); got == 0 {
		killTimer.Call(hwnd, timer)
		return fmt.Errorf("PeekMessageW did not observe timer dispatch: %v", err)
	}
	translateMessage.Call(uintptr(unsafe.Pointer(&m)))
	dispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	killTimer.Call(hwnd, timer)
	if ok, _, err := sendMessage.Call(label, wmSetText, 0, uintptr(unsafe.Pointer(timerSaved))); ok == 0 {
		return fmt.Errorf("SendMessageW label timer state failed: %v", err)
	}
	if ok, _, err := redrawWindow.Call(hwnd, 0, 0, rdwInvalidate|rdwAllChildren|rdwUpdateNow); ok == 0 {
		return fmt.Errorf("RedrawWindow failed: %v", err)
	}
	return nil
}
