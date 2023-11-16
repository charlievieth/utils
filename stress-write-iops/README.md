# stress-write-iops

stress-write-iops stresses disk I/O and prints latency statistics.

## Notes

* The name of the temp directory used on each run is printed at
  program startup - in case you need to manually delete it.
* CTRL-C will cause stress-write-iops to stop and remove it's
  temporary files. Pressing CTRL-C again will immediately halt it
* Each writer uses 4MiB of disk space and a recovery file
  `/tmp/recovery.dat` is created at startup so that if you run out
  of disk space you can stop `stress-write-iops` and delete it file
  to recover enough disk space to cleanup the temp directory used.

Run `./stress-write-iops -help` for a description of commands.

The `./run.bash` script is provided as a helper and on Linux it
will flush the page cache and dentries/inodes caches every second.
On Linux it will also write the output of `iostat` at 1 and 10
seconds to `./iostat_1s.log` and `./iostat_10s.log`.

## Example

Run stress-write-iops with 256 writers, pausing 1ns between writes,
printing disk stats every second, and syncing files after writes.
```sh
./run.bash  -n 256 -d 1ns -disk-stat-int 1s -sync
```

Same as above but
```sh
./stress-write-iops  -n 256 -d 1ns -disk-stat-int 1s -sync
```
