#!/usr/bin/env bash

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
# Register exit event handler.
trap on_exit EXIT

apt install libpcap-dev