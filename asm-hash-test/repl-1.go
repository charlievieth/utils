package main

import (
	"crypto/rand"
	"fmt"
	"runtime"
	"time"
	"unsafe"
)

// TEXT runtime·aeshash64(SB),NOSPLIT,$0-24
// 	MOVQ	p+0(FP), AX	// ptr to data
// 	MOVQ	h+8(FP), X0	// seed
// 	PINSRQ	$1, (AX), X0	// data
// 	AESENC	runtime·aeskeysched+0(SB), X0
// 	AESENC	runtime·aeskeysched+16(SB), X0
// 	AESENC	runtime·aeskeysched+32(SB), X0
// 	MOVQ	X0, ret+16(FP)
// 	RET

const PtrSize = 4 << (^uintptr(0) >> 63) // 8

const hashRandomBytes = PtrSize / 4 * 64

// used in asm_{386,amd64}.s to seed the hash function
var aeskeysched [hashRandomBytes]byte

func init() {
	getRandomData(aeskeysched[:])
}

func getRandomData(r []byte) {
	n, err := rand.Read(r)
	if err != nil {
		panic(err)
	}
	extendRandom(r, n)
}

// extendRandom extends the random numbers in r[:n] to the whole slice r.
// Treats n<0 as n==0.
func extendRandom(r []byte, n int) {
	if n < 0 {
		n = 0
	}
	for n < len(r) {
		// Extend random bits using hash function & time seed
		w := n
		if w > 16 {
			w = 16
		}
		h := memhash(unsafe.Pointer(&r[n-w]), uintptr(time.Now().Nanosecond()), uintptr(w))
		for i := 0; i < PtrSize && n < len(r); i++ {
			r[n] = byte(h)
			n++
			h >>= 8
		}
	}
}

func aeshash(p unsafe.Pointer, h, s uintptr) uintptr
func aeshash32(p unsafe.Pointer, h uintptr) uintptr
func aeshash64(p unsafe.Pointer, h uintptr) uintptr
func aeshashstr(p unsafe.Pointer, h uintptr) uintptr

func memhash(p unsafe.Pointer, seed, s uintptr) uintptr {
	if runtime.GOARCH == "amd64" && runtime.GOOS != "nacl" {
		return aeshash(p, seed, s)
	}
	panic(runtime.GOARCH + "_" + runtime.GOOS)
}

// {150, 70, 72, 75, 247, 172, 125, 63, 176, 191, 0, 219, 250, 206, 96, 2, 180, 144, 25, 57, 173, 144, 70, 164, 230, 194, 43, 192, 167, 128, 211, 223, 110, 78, 93, 52, 98, 0, 204, 82, 76, 226, 203, 220, 227, 69, 131, 30, 119, 233, 40, 138, 54, 21, 21, 139, 21, 21, 59, 30, 205, 234, 214, 16, 39, 12, 60, 65, 5, 252, 68, 143, 157, 184, 182, 14, 137, 133, 204, 235, 80, 244, 188, 149, 31, 96, 255, 8, 220, 119, 15, 23, 107, 119, 143, 14, 9, 123, 176, 28, 39, 92, 68, 57, 210, 69, 111, 105, 53, 218, 114, 203, 43, 245, 206, 203, 125, 239, 0, 86, 9, 194, 247, 138, 209, 145, 242, 98}

func main() {
	fmt.Print("{")
	fmt.Print(aeskeysched[0])
	for i := 1; i < len(aeskeysched); i++ {
		fmt.Printf(", %d", aeskeysched[i])
	}
	fmt.Println("}")
	// fmt.Println(aeskeysched)
}
