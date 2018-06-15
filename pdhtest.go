package main

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

const (
	PDH_MORE_DATA = 0x800007D2
	ERROR_SUCCESS = 0
)

var (
	pdh_ParseCounterPathW *syscall.Proc
	libpdhDll *syscall.DLL
)

type PDH_COUNTER_PATH_ELEMENTS struct {
	MachineName *uint16
	ObjectName *uint16
	InstanceName *uint16
	ParentInstance *uint16
	InstanceIndex uint32
	CounterName *uint16
}

type pdhCounter struct{
	Path             	string `yaml:"Path"`   // The path to the PDH counter to be collected
	ExcludeInstances 	[]string // A list of PDH instances to be excluded from collection
	MachineName string
	ObjectName string
	InstanceName string
	ParentInstance string
	InstanceIndex uint32
	CounterName string
}

func (p *pdhCounter) Print() string {
	return fmt.Sprintf("M: %s, O: %s, I: %s, C: %s", p.MachineName, p.ObjectName, p.InstanceName, p.CounterName)
}

func init() {
	// Library
	libpdhDll = syscall.MustLoadDLL("pdh.dll")
	pdh_ParseCounterPathW = libpdhDll.MustFindProc("PdhParseCounterPathW")
}

func main() {
	paths := []string {
		`\Memory\% Committed Bytes In Use`,
		`\Memory\Available Bytes`,
		`\PhysicalDisk(*)\Avg. Disk Bytes/Transfer`,
		`\PhysicalDisk(*)\Avg. Disk sec/Transfer`,
		`\PhysicalDisk(*)\Disk Bytes/sec`,
		`\PhysicalDisk(*)\Disk Transfers/sec`,
		`\Processor(*)\% Processor Time`,
	}

	for _, path := range paths {
		c, err := CounterPath(path)
		if err != nil {
			panic(err)
		} else {
			fmt.Println(c.Print())
		}
	}
}

func CounterPath(path string) (*pdhCounter, error) {
	var c PDH_COUNTER_PATH_ELEMENTS
	var b uint32
	if ret := PdhParseCounterPath(path, nil, &b); ret == PDH_MORE_DATA {
		if ret := PdhParseCounterPath(path, &c, &b); ret == ERROR_SUCCESS {
			p := pdhCounter{
				Path: path,
				MachineName: UTF16PtrToString(c.MachineName)[2:],
				ObjectName: UTF16PtrToString(c.ObjectName),
				InstanceName: UTF16PtrToString(c.InstanceName),
				ParentInstance: UTF16PtrToString(c.ParentInstance),
				InstanceIndex: c.InstanceIndex,
				CounterName: UTF16PtrToString(c.CounterName),
			}
			return &p, nil
		} else {
			// Failed to parse counter
			// Possible error codes: PDH_INVALID_ARGUMENT, PDH_INVALID_PATH, PDH_MEMORY_ALLOCATION_FAILURE
			return nil, errors.New("failed to create PdhCounter from " + path + " -> " + fmt.Sprintf("%x", ret))
		}
	} else {
		// Failed to obtain buffer info
		// Possible error codes: PDH_INVALID_ARGUMENT, PDH_INVALID_PATH, PDH_MEMORY_ALLOCATION_FAILURE
		return nil, errors.New("failed to obtain buffer info for PdhCounter from " + path + " -> " + fmt.Sprintf("%x", ret))
	}
}

func PdhParseCounterPath(szFullPathBuffer string, pCounterPathElements *PDH_COUNTER_PATH_ELEMENTS, pdwBufferSize *uint32) uint32 {
	ptxt, _ := syscall.UTF16PtrFromString(szFullPathBuffer)

	ret, _, _ := pdh_ParseCounterPathW.Call(
		uintptr(unsafe.Pointer(ptxt)),
		uintptr(unsafe.Pointer(pCounterPathElements)),
		uintptr(unsafe.Pointer(pdwBufferSize)),
		0)

	return uint32(ret)
}

func UTF16PtrToString(s *uint16) string {
	if s == nil {
		return ""
	}
	return syscall.UTF16ToString((*[1 << 29]uint16)(unsafe.Pointer(s))[0:])
}
