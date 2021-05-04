# vim: filetype=sh

export CHIST_CLIENT_CMD=/Users/cvieth/go/src/github.com/charlievieth/utils/histdb/cmd/chist-client/chist-client

export HISTDB_ENABLED=0

# WARN: using this for testing
histdb-enable() {
    export HISTDB_ENABLED=1
}

# WARN: using this for testing
histdb-disable() {
    export HISTDB_ENABLED=0
}

__histdb_generate_session_id() {
    \head -c8 /dev/urandom | \od -t u8 | \awk '{print $2}' | \head -n1
}

__histdb_session_id() {
    if [[ -z $HISTDB_SESSION_ID ]]; then
        local id
        id=$(__histdb_generate_session_id)
        export HISTDB_SESSION_ID=$id
    fi
    echo "$HISTDB_SESSION_ID"
}

__histdb_preexec() {
    if (( HISTDB_ENABLED )); then
        export __histdb_last_cmd=("$@")
    fi
    # echo "preexec: $? -- ${__histdb_last_cmd[@]}"
}

__histdb_precmd() {
    if (( ! HISTDB_ENABLED )); then
        return
    fi
    # only run if the last command is set
    if [[ -v __histdb_last_cmd ]] && [[ -v __bp_last_ret_value ]]; then
        local session status_code
        session="$(__histdb_session_id)"
        status_code=$__bp_last_ret_value
        "$CHIST_CLIENT_CMD" --session="$session" --status-code="$status_code" -- "${__histdb_last_cmd[@]}"

        # echo "cmd:" "$ret" "--" "${__histdb_last_cmd[@]}"
    fi
    unset __histdb_last_cmd # always unset
}

if [[ -v preexec_functions ]]; then
    preexec_functions+=(__histdb_preexec)
else
    preexec_functions=(__histdb_preexec)
fi

if [[ -v precmd_functions ]]; then
    precmd_functions+=(__histdb_precmd)
else
    precmd_functions=(__histdb_precmd)
fi


# __histdb_precmd() {
#     local status_code=$?
#     local session_id="$HISTDB_SESSION_ID"
#
#     "$CHIST_CLIENT_CMD" --session="$session_id" --status-code="$status_code"
# }
#
# export PROMPT_COMMAND="/bin/echo 'FOO'"
