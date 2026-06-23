package main
import (
	"fmt"
	"strings"
	"sync"
	"syscall"
	"unsafe"
	"golang.org/x/sys/windows"
)
const entityListStride = 112
var (
	ntdll               = syscall.NewLazyDLL("ntdll.dll")
	ntReadVirtualMemory = ntdll.NewProc("NtReadVirtualMemory")
	readBufferPool = sync.Pool{
		New: func() any {
			buf := make([]byte, 4096)
			return &buf
		},
	}
)
func getModuleBaseAddress(pid int, moduleName string) (uintptr, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPMODULE|windows.TH32CS_SNAPMODULE32, uint32(pid))
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(snapshot)
	var me32 windows.ModuleEntry32
	me32.Size = uint32(unsafe.Sizeof(me32))
	if err := windows.Module32First(snapshot, &me32); err != nil {
		return 0, fmt.Errorf("Module32First failed: %w", err)
	}
	for {
		if strings.EqualFold(windows.UTF16ToString(me32.Module[:]), moduleName) {
			return uintptr(me32.ModBaseAddr), nil
		}
		if err := windows.Module32Next(snapshot, &me32); err != nil {
			break
		}
	}
	return 0, fmt.Errorf("module %q not found", moduleName)
}

func getProcessHandle(pid int) (windows.Handle, error) {
	return windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, uint32(pid))
}

func findProcessId(name string) (int, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(snapshot)
	var process windows.ProcessEntry32
	process.Size = uint32(unsafe.Sizeof(process))
	if err := windows.Process32First(snapshot, &process); err != nil {
		return 0, fmt.Errorf("Process32First failed: %w", err)
	}
	for {
		if strings.EqualFold(windows.UTF16ToString(process.ExeFile[:]), name) {
			return int(process.ProcessID), nil
		}
		if err := windows.Process32Next(snapshot, &process); err != nil {
			break
		}
	}
	return 0, fmt.Errorf("process %q not found", name)
}

func readMemory(process windows.Handle, address uintptr, buffer []byte) error {
	if len(buffer) == 0 {
		return nil
	}
	var bytesRead uintptr
	status, _, _ := ntReadVirtualMemory.Call(
		uintptr(process), address, uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)), uintptr(unsafe.Pointer(&bytesRead)),
	)
	if status != 0 {
		return fmt.Errorf("NtReadVirtualMemory failed: 0x%X", status)
	}
	if bytesRead != uintptr(len(buffer)) {
		return fmt.Errorf("incomplete read: expected %d, got %d", len(buffer), bytesRead)
	}
	return nil
}

func withPooledBuffer(size int, fn func([]byte) error) error {
	poolBuf := readBufferPool.Get().(*[]byte)
	buf := *poolBuf
	if cap(buf) < size {
		buf = make([]byte, size)
	} else {
		buf = buf[:size]
	}
	err := fn(buf)
	*poolBuf = buf
	readBufferPool.Put(poolBuf)
	return err
}

func readInto(process windows.Handle, address uintptr, dst unsafe.Pointer, size int) error {
	return withPooledBuffer(size, func(buf []byte) error {
		if err := readMemory(process, address, buf); err != nil {
			return err
		}
		copy(unsafe.Slice((*byte)(dst), size), buf)
		return nil
	})
}

func readAt(process windows.Handle, address uintptr, size int) ([]byte, error) {
	out := make([]byte, size)
	if err := readInto(process, address, unsafe.Pointer(&out[0]), size); err != nil {
		return nil, err
	}
	return out, nil
}

func readSafe(process windows.Handle, address uintptr, value interface{}) error {
	switch v := value.(type) {
	case *uint8:
		return readInto(process, address, unsafe.Pointer(v), 1)
	case *bool:
		return readInto(process, address, unsafe.Pointer(v), 1)
	case *int32:
		return readInto(process, address, unsafe.Pointer(v), 4)
	case *uint32:
		return readInto(process, address, unsafe.Pointer(v), 4)
	case *uintptr:
		return readInto(process, address, unsafe.Pointer(v), int(unsafe.Sizeof(uintptr(0))))
	case *Vector3:
		return readInto(process, address, unsafe.Pointer(v), 12)
	case *[16]float32:
		return readInto(process, address, unsafe.Pointer(v), 64)
	case *[]byte:
		return readInto(process, address, unsafe.Pointer(&(*v)[0]), len(*v))
	default:
		return fmt.Errorf("unsupported read type: %T", value)
	}
}

func readPtr(process windows.Handle, address uintptr) (uintptr, error) {
	var ptr uintptr
	if err := readSafe(process, address, &ptr); err != nil {
		return 0, err
	}
	return ptr, nil
}

func readEntityFromList(process windows.Handle, entityList uintptr, handle uint32) (uintptr, error) {
	if handle == 0 || entityList == 0 {
		return 0, fmt.Errorf("invalid handle or entity list")
	}
	listEntry, err := readPtr(process, entityList+uintptr(0x8*((handle&0x7FFF)>>9)+16))
	if err != nil || listEntry == 0 {
		return 0, err
	}
	return readPtr(process, listEntry+uintptr(entityListStride)*uintptr(handle&0x1FF))
}

func readController(process windows.Handle, entityList uintptr, index int) (uintptr, error) {
	listEntry, err := readPtr(process, entityList+uintptr((8*(index&0x7FFF)>>9)+16))
	if err != nil || listEntry == 0 {
		return 0, err
	}
	return readPtr(process, listEntry+uintptr(entityListStride)*uintptr(index&0x1FF))
}
