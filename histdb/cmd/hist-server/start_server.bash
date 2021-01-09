#!/usr/bin/env bash

set -euo pipefail

######################################################################
# Using daemon program

# TODO: use something like HISTDBHOME to override default histdb dir
# HISTDBHOME=~/.local/share/histdb

if ! hash hist-server &>/dev/null; then
    echo >&2 'error: hist-server not installed'
    exit 1
fi

share_dir=~/.local/share/histdb
log_dir="$share_dir/logs"

[ -d "$share_dir" ] || mkdir -p "$share_dir"
[ -d "$log_dir" ] || mkdir -p "$log_dir"

base_flags=(
    --pidfiles="$share_dir"
    --name=hist-server
)

# flags=(
#     --noconfig
#     --inherit
#     --name=hist-server
#     --dbglog="$log_dir/daemon.dbg.log"
#     --errlog="$log_dir/daemon.err.log"
#     --stdout="$log_dir/server.out.log"
#     --stderr="$log_dir/server.err.log"
#     "${base_flags[@]}"
# )
# if ! daemon "${flags[@]}" -- hist-server; then
#     echo >&2 'error: running daemon to start hist-server'
#     exit 1
# fi

# if ! daemon "${base_flags[@]}" --running; then
#     echo >&2 'error: starting hist-server'
#     exit 1
# fi

# WARN
_command_name() { echo "$0" | grep -oE '[^/]+$'; }

die() {
    echo -e "$(_command_name):" "$@" >&2
    exit 1
}

usage() {
    echo "$(_command_name): COMMAND"
    echo ''
    echo 'Commands:'
    echo '    restart: restart the histdb server (if running)'
    echo '    running: check if the histdb server is running'
    echo '    stop: stop the histdb server (if running)'
    echo '    start (default): start the histdb if not running'
}

_invalid_cmd() {
    local cmd="$1"
    local name
    name="$"
}

cmd="${1:-start}"
case "$cmd" in
    restart)
        exec daemon "${base_flags[@]}" --restart
        ;;
    running)
        exec daemon "${base_flags[@]}" --running
        ;;
    stop)
        exec daemon "${base_flags[@]}" --stop
        ;;
    start)
        flags=(
            --noconfig
            --inherit
            --name=hist-server
            --dbglog="$log_dir/daemon.dbg.log"
            --errlog="$log_dir/daemon.err.log"
            --stdout="$log_dir/server.out.log"
            --stderr="$log_dir/server.err.log"
            "${base_flags[@]}"
        )
        if ! daemon "${flags[@]}" -- hist-server; then
            die 'error: running daemon to start hist-server'
        fi

        if ! daemon "${base_flags[@]}" --running; then
            die 'error: starting hist-server'
        fi
        ;;
    help | usage)
        usage
        ;;
    *)
        die "unrecognized option '$cmd'\nTry 'help' for more information" >&2
        ;;
esac

######################################################################
# Old: using bash as a crappy daemon mgr

# PID_FILE=~/.local/share/histdb/pid
#
# if ! hash hist-server &>/dev/null; then
#     echo >&2 'error: hist-server not installed'
#     exit 1
# fi
#
# if [[ -r "$PID_FILE" ]]; then
#     PID="$(head -n1 "$PID_FILE")"
#     if echo "$PID" | \grep --quiet -E '^[0-9]+$'; then
#         if kill -0 "$PID" &>/dev/null; then
#             exit 0 # proc running
#         fi
#     else
#         rm "$PID_FILE" # invalid
#     fi
# fi
#
# mkdir -p "$(dirname "$PID_FILE")"
#
# hist-server &
#
# SRV_PID=$!
# echo "$SRV_PID" >| "$PID_FILE"
# trap 'rm $PID_FILE; kill -15 $SRV_PID' EXIT
#
# wait $SRV_PID
