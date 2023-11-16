#!/usr/bin/env bash

set -euo pipefail

if ! command -v >/dev/null; then
    echo 'error: iostat not installed!'
    exit 1
fi

# Make sure we can run sudo
sudo /usr/bin/true

# On Linux periodically drop the file system cache
# to greatly increase pressure on the disk.
if [[ "$(uname -s)" == Linux ]]; then
    (
        # 1: free page cache
        # 2: free reclaimable slab objects (includes dentries and inodes)
        # 3: free slab objects and pagecache
        MODE=3
        while true; do
            sudo sh -c "echo ${MODE} > /proc/sys/vm/drop_caches"
            sleep 1
        done
    ) &
    # Write iostat output
    (
        # 1 second interval
        LC_TIME=en_US.UTF-8 iostat -m -t -x 1 &>./iostat_1s.log
    ) &
    (
        # 10 second interval
        LC_TIME=en_US.UTF-8 iostat -m -t -x 10 &>./iostat_10s.log
    ) &
fi

# kill jobs on exit
trap '[ -n "$(jobs -p)" ] && kill -TERM $(jobs -p) 2>/dev/null' EXIT

go build -buildvcs=false && ./stress-write-iops "$@"
