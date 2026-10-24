//go:build windows

package tray

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")

	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procLoadIconW           = user32.NewProc("LoadIconW")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procPostMessageW        = user32.NewProc("PostMessageW")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
)

const (
	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004

	WM_DESTROY       = 0x0002
	WM_CLOSE         = 0x0010
	WM_USER          = 0x0400
	WM_TRAY_MSG      = WM_USER + 101
	WM_COMMAND       = 0x0111
	WM_LBUTTONUP     = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONUP     = 0x0205
	WM_CONTEXTMENU   = 0x007B
	WM_NULL          = 0x0000

	MF_STRING    = 0x00000000
	MF_SEPARATOR = 0x00000800

	TPM_BOTTOMALIGN = 0x0020
	TPM_RIGHTBUTTON = 0x0002

	IDI_APPLICATION = 32512

	cmdOpen     = 2001
	cmdPauseAll = 2002
	cmdQuit     = 2003
)

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type POINT struct {
	X, Y int32
}

type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type NOTIFYICONDATAW struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UTimeoutOrVer    uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         windows.GUID
	HBalloonIcon     uintptr
}

var (
	activeTrayMu sync.Mutex
	activeTray   *winTray
)

type winTray struct {
	callbacks Callbacks
	hwnd      uintptr
	nid       NOTIFYICONDATAW
	readyChan chan error
	stopOnce  sync.Once
}

// NewTray constructs a Windows native system tray manager.
func NewTray(callbacks Callbacks) Tray {
	return &winTray{
		callbacks: callbacks,
		readyChan: make(chan error, 1),
	}
}

func (t *winTray) Start() error {
	activeTrayMu.Lock()
	activeTray = t
	activeTrayMu.Unlock()

	go t.runMessageLoop()

	return <-t.readyChan
}

func (t *winTray) Stop() {
	t.stopOnce.Do(func() {
		if t.hwnd != 0 {
			procPostMessageW.Call(t.hwnd, WM_CLOSE, 0, 0)
		}
	})
}

func stringToUTF16Fixed128(s string) [128]uint16 {
	var result [128]uint16
	u16 := windows.StringToUTF16(s)
	for i := 0; i < len(u16) && i < 127; i++ {
		result[i] = u16[i]
	}
	return result
}

func (t *winTray) runMessageLoop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	className, _ := windows.UTF16PtrFromString("CandakTrayWindowClass")

	var wc WNDCLASSEXW
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.LpfnWndProc = syscall.NewCallback(trayWndProc)
	wc.HInstance = hInstance
	wc.LpszClassName = className

	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	windowName, _ := windows.UTF16PtrFromString("CandakTrayWindow")
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		0,
		0, 0, 0, 0,
		0,
		0,
		hInstance,
		0,
	)

	if hwnd == 0 {
		t.readyChan <- fmt.Errorf("failed to create tray message window")
		return
	}
	t.hwnd = hwnd

	// Load Icon: try application icon from PE resources (ID 1), fallback to standard IDI_APPLICATION
	hIcon, _, _ := procLoadIconW.Call(hInstance, uintptr(1))
	if hIcon == 0 {
		hIcon, _, _ = procLoadIconW.Call(0, uintptr(IDI_APPLICATION))
	}

	t.nid = NOTIFYICONDATAW{
		CbSize:           uint32(unsafe.Sizeof(t.nid)),
		HWnd:             hwnd,
		UID:              1,
		UFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
		UCallbackMessage: WM_TRAY_MSG,
		HIcon:            hIcon,
		SzTip:            stringToUTF16Fixed128("Candak Download Manager"),
	}

	ret, _, _ := procShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&t.nid)))
	if ret == 0 {
		t.readyChan <- fmt.Errorf("failed to register system tray icon")
		return
	}

	t.readyChan <- nil

	var msg MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func trayWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	activeTrayMu.Lock()
	t := activeTray
	activeTrayMu.Unlock()

	if t == nil {
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}

	switch msg {
	case WM_TRAY_MSG:
		switch lParam {
		case WM_LBUTTONUP, WM_LBUTTONDBLCLK:
			if t.callbacks.OnOpen != nil {
				go t.callbacks.OnOpen()
			}
			return 0
		case WM_RBUTTONUP, WM_CONTEXTMENU:
			var pt POINT
			procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
			hMenu, _, _ := procCreatePopupMenu.Call()
			if hMenu != 0 {
				openStr, _ := windows.UTF16PtrFromString("Open Candak")
				pauseStr, _ := windows.UTF16PtrFromString("Pause All")
				quitStr, _ := windows.UTF16PtrFromString("Exit")

				procAppendMenuW.Call(hMenu, MF_STRING, cmdOpen, uintptr(unsafe.Pointer(openStr)))
				procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
				procAppendMenuW.Call(hMenu, MF_STRING, cmdPauseAll, uintptr(unsafe.Pointer(pauseStr)))
				procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
				procAppendMenuW.Call(hMenu, MF_STRING, cmdQuit, uintptr(unsafe.Pointer(quitStr)))

				procSetForegroundWindow.Call(hwnd)
				procTrackPopupMenu.Call(hMenu, TPM_BOTTOMALIGN|TPM_RIGHTBUTTON, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
				procPostMessageW.Call(hwnd, WM_NULL, 0, 0)
				procDestroyMenu.Call(hMenu)
			}
			return 0
		}
	case WM_COMMAND:
		switch wParam {
		case cmdOpen:
			if t.callbacks.OnOpen != nil {
				go t.callbacks.OnOpen()
			}
			return 0
		case cmdPauseAll:
			if t.callbacks.OnPauseAll != nil {
				go t.callbacks.OnPauseAll()
			}
			return 0
		case cmdQuit:
			if t.callbacks.OnQuit != nil {
				go t.callbacks.OnQuit()
			}
			return 0
		}
	case WM_CLOSE:
		procDestroyWindow.Call(hwnd)
		return 0
	case WM_DESTROY:
		procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&t.nid)))
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}
