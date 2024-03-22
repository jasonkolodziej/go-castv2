#!/usr/bin/env bash

TERM=xterm-256color
#? Use:
#? DEBUG echo "I am debugging"
#? DEBUG set -x #turn on | +x # off
function DEBUG()
{
 [ "$_DEBUG" == "on" ] &&  $@
}

# Exit event handler
function on_exit() {
    tput cnorm # Show cursor. You need this if animation is used.
    # i.e. clean-up code here
    exit 0 # Exit gracefully.
}

# Put this line at the beginning of your script (after functions used by event handlers).
trap on_exit EXIT

# echo "http://${2}:${3}/devices/${4}/${1}"


curl -s -o /dev/null -w "%{http_code}" -X GET "http://${2}:${3}/devices/${4}/${1}"
on_exit