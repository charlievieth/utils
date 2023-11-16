//go:build linux
// +build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"syscall"
)

func Fallocate(filename string, size int64) error {
	if size < 0 {
		return errors.New("fallocate: negative size")
	}
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := syscall.Fallocate(int(f.Fd()), 0, 0, size); err != nil {
		return fmt.Errorf("fallocate: %w", err)
	}
	return f.Close()
}

// drop_caches
// ===========
//
// Writing to this will cause the kernel to drop clean caches, as well as
// reclaimable slab objects like dentries and inodes.  Once dropped, their
// memory becomes free.
//
// To free pagecache::
//
//	echo 1 > /proc/sys/vm/drop_caches
//
// To free reclaimable slab objects (includes dentries and inodes)::
//
//	echo 2 > /proc/sys/vm/drop_caches
//
// To free slab objects and pagecache::
//
//	echo 3 > /proc/sys/vm/drop_caches
//
// This is a non-destructive operation and will not free any dirty objects.
// To increase the number of objects freed by this operation, the user may run
// `sync` prior to writing to /proc/sys/vm/drop_caches.  This will minimize the
// number of dirty objects on the system and create more candidates to be
// dropped.
//
// This file is not a means to control the growth of the various kernel caches
// (inodes, dentries, pagecache, etc...)  These objects are automatically
// reclaimed by the kernel when memory is needed elsewhere on the system.
//
// Use of this file can cause performance problems.  Since it discards cached
// objects, it may cost a significant amount of I/O and CPU to recreate the
// dropped objects, especially if they were under heavy use.  Because of this,
// use outside of a testing or debugging environment is not recommended.
//
// https://github.com/torvalds/linux/blob/61d325dcbc05d8fef88110d35ef7776f3ac3f68b/Documentation/admin-guide/sysctl/vm.rst?plain=1#L227-L267
func DropCaches(sync, pagecache, dentsInodes bool) error {
	if sync {
		syscall.Sync()
	}
	if !pagecache && !dentsInodes {
		return nil // TODO: maybe error
	}
	var mode int64
	if pagecache {
		mode |= 1
	}
	if dentsInodes {
		mode |= 2
	}
	return os.WriteFile("/proc/sys/vm/drop_caches", strconv.AppendInt(nil, mode, 10), 0)
}
