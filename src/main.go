package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Config struct {
	Mapping map[string][]string `json:"keys:mapping"`
}

var (
	cfgMu   sync.RWMutex
	reflect map[uint16]uint16
	pressed [256]bool
	hook    uintptr
)

const (
	WH_KEYBOARD_LL     = 13
	WM_KEYDOWN         = 0x0100
	WM_KEYUP           = 0x0101
	WM_SYSKEYDOWN      = 0x0104
	WM_SYSKEYUP        = 0x0105
	LLKHF_INJECTED     = 0x10
	INPUT_KEYBOARD     = 1
	KEYEVENTF_KEYUP    = 0x0002
	KEYEVENTF_SCANCODE = 0x0008
	MAPVK_VK_TO_VSC    = 0
)

type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type INPUT struct {
	Type uint32
	Ki   KEYBDINPUT
	_    [8]byte
}

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	procSendInput        = user32.NewProc("SendInput")
	procSetWindowsHookEx = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx   = user32.NewProc("CallNextHookEx")
	procUnhook           = user32.NewProc("UnhookWindowsHookEx")
	procGetMessage       = user32.NewProc("GetMessageW")
	procMapVirtualKey    = user32.NewProc("MapVirtualKeyW")
)

//go:nocheckptr
func ptrFromUintptr(p uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&p))
}

var nameToVK = map[string]uint16{
	"space": 0x20, "enter": 0x0D, "esc": 0x1B, "tab": 0x09,
	"shift": 0x10, "ctrl": 0x11, "alt": 0x12,
	"f1": 0x70, "f2": 0x71, "f3": 0x72, "f4": 0x73,
	"f5": 0x74, "f6": 0x75, "f7": 0x76, "f8": 0x77,
	"f9": 0x78, "f10": 0x79, "f11": 0x7A, "f12": 0x7B,
}

func toVK(s string) (uint16, bool) {
	if len(s) == 1 {
		c := s[0]
		switch {
		case c >= 'a' && c <= 'z':
			return uint16(c - 'a' + 'A'), true
		case c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			return uint16(c), true
		}
	}
	v, ok := nameToVK[s]
	return v, ok
}

func vkToScan(vk uint16) uint16 {
	r, _, _ := procMapVirtualKey.Call(uintptr(vk), MAPVK_VK_TO_VSC)
	return uint16(r)
}

func buildReflect(m map[string][]string) map[uint16]uint16 {
	t := make(map[uint16]uint16, len(m))
	for logical, members := range m {
		lv, ok := toVK(logical)
		if !ok {
			continue
		}
		for _, mem := range members {
			mv, ok := toVK(mem)
			if !ok {
				continue
			}
			if mv != lv {
				t[mv] = lv
			}
		}
	}
	return t
}

func sendKey(vk uint16, up bool) {
	var flags uint32 = KEYEVENTF_SCANCODE
	if up {
		flags |= KEYEVENTF_KEYUP
	}
	in := INPUT{
		Type: INPUT_KEYBOARD,
		Ki: KEYBDINPUT{
			WScan:   vkToScan(vk),
			DwFlags: flags,
		},
	}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

func hookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= 0 {
		kb := (*KBDLLHOOKSTRUCT)(ptrFromUintptr(lParam))
		vk := uint16(kb.VkCode)

		if kb.Flags&LLKHF_INJECTED == 0 {
			cfgMu.RLock()
			target, ok := reflect[vk]
			cfgMu.RUnlock()

			if ok {
				switch wParam {
				case WM_KEYDOWN, WM_SYSKEYDOWN:
					if !pressed[vk] {
						pressed[vk] = true
						sendKey(target, false)
					}
					return 1
				case WM_KEYUP, WM_SYSKEYUP:
					pressed[vk] = false
					sendKey(target, true)
					return 1
				}
			}
		}
	}
	ret, _, _ := procCallNextHookEx.Call(hook, uintptr(nCode), wParam, lParam)
	return ret
}

func loadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return err
	}
	table := buildReflect(c.Mapping)

	cfgMu.Lock()
	reflect = table
	cfgMu.Unlock()
	return nil
}

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	exe, err := os.Executable()
	if err != nil {
		return
	}
	cfgPath := filepath.Join(filepath.Dir(exe), "json", "keymacro.json")
	if err := loadConfig(cfgPath); err != nil {
		return
	}
	if len(reflect) == 0 {
		return
	}

	hook, _, _ = procSetWindowsHookEx.Call(
		WH_KEYBOARD_LL,
		windows.NewCallback(hookProc),
		0, 0,
	)
	if hook == 0 {
		return
	}
	defer procUnhook.Call(hook)

	var msg MSG
	for {
		r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if r == 0 || r == ^uintptr(0) {
			break
		}
	}
}