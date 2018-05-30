package main

import (
	"fmt"
	"unsafe"

	"syscall"
)

type LARGE_INTEGER struct {
	LowPart  int
	HighPart int64
}

func main() {
	fmt.Println(syscall.EWINDOWS, uint(syscall.EWINDOWS))
}

func XX() {
	kernel32DLL := syscall.MustLoadDLL("Kernel32.dll")
	proc := kernel32DLL.MustFindProc("QueryPerformanceCounter")
	var count int64
	if _, _, err := proc.Call((uintptr)(unsafe.Pointer(&count))); err != nil {
		fmt.Println("Error:", err)
	}
	fmt.Println("Count:", count)
}
