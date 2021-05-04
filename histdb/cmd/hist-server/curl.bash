#!/bin/sh

set -e

curl --unix-socket /Users/cvieth/.local/share/histdb/socket/sock.sock http://localhost/"$1"
