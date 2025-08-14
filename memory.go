package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ntdll               = syscall.NewLazyDLL("ntdll.dll")
	ntReadVirtualMemory = ntdll.NewProc("NtReadVirtualMemory")
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
	return 0, fmt.Errorf("module not found")
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
	return 0, fmt.Errorf("process not found")
}

func readMemoryNt(process windows.Handle, address uintptr, buffer []byte) error {
	var bytesRead uintptr
	size := uintptr(len(buffer))
	status, _, _ := ntReadVirtualMemory.Call(
		uintptr(process), address, uintptr(unsafe.Pointer(&buffer[0])),
		size, uintptr(unsafe.Pointer(&bytesRead)),
	)
	if status != 0 {
		return fmt.Errorf("NtReadVirtualMemory failed with status: 0x%X", status)
	}
	if bytesRead != size {
		return fmt.Errorf("incomplete read: expected %d bytes, got %d", size, bytesRead)
	}
	return nil
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
	buffer := make([]byte, size)
	if err := readMemoryNt(process, address, buffer); err != nil {
		return err
	}
	reader := bytes.NewReader(buffer)
	switch v := value.(type) {
	case *[]byte:
		copy(*v, buffer)
		return nil
	case *uintptr:
		var u64 uint64
		if err := binary.Read(reader, binary.LittleEndian, &u64); err != nil {
			return err
		}
		*v = uintptr(u64)
		return nil
	default:
		return binary.Read(reader, binary.LittleEndian, value)
	}
}