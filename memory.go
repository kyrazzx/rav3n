package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const entityListStride = 120

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

func readAt(process windows.Handle, address uintptr, size int) ([]byte, error) {
	poolBuf := readBufferPool.Get().(*[]byte)
	buf := *poolBuf
	if cap(buf) < size {
		buf = make([]byte, size)
	} else {
		buf = buf[:size]
	}
	if err := readMemory(process, address, buf); err != nil {
		readBufferPool.Put(poolBuf)
		return nil, err
	}
	out := make([]byte, size)
	copy(out, buf)
	*poolBuf = buf
	readBufferPool.Put(poolBuf)
	return out, nil
}

func readSafe(process windows.Handle, address uintptr, value interface{}) error {
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("value must be a pointer")
	}
	elem := val.Elem()
	size := int(elem.Type().Size())
	if size == 0 {
		if elem.Type().Kind() == reflect.Slice {
			size = elem.Cap()
			if size == 0 {
				return fmt.Errorf("cannot read into a slice with 0 capacity")
			}
		} else {
			return fmt.Errorf("unsupported type with size 0: %T", value)
		}
	}

	poolBuf := readBufferPool.Get().(*[]byte)
	buf := *poolBuf
	if cap(buf) < size {
		buf = make([]byte, size)
	} else {
		buf = buf[:size]
	}
	if err := readMemory(process, address, buf); err != nil {
		readBufferPool.Put(poolBuf)
		return err
	}

	reader := bytes.NewReader(buf)
	switch v := value.(type) {
	case *[]byte:
		copy(*v, buf)
	case *uintptr:
		var u64 uint64
		if err := binary.Read(reader, binary.LittleEndian, &u64); err != nil {
			readBufferPool.Put(poolBuf)
			return err
		}
		*v = uintptr(u64)
	default:
		if err := binary.Read(reader, binary.LittleEndian, value); err != nil {
			readBufferPool.Put(poolBuf)
			return err
		}
	}
	*poolBuf = buf
	readBufferPool.Put(poolBuf)
	return nil
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
