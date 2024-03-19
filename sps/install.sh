#!/usr/bin/env bash
_DEBUG="on"
PKG_OWN="mikebrady"
PKG1="nqptp"
PKG2="shairport-sync"

OTHER_PKGS="ffmpeg"
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


function prepare_for_install () {
    if (( $(id -u) != 0 )); then
        echo "I'm not root"
        on_exit "please run with privileges"
        echo "$?"
    fi
    sudo apt update
    sudo apt upgrade
    sudo apt install --no-install-recommends \
        build-essential git autoconf automake \
        libtool libglib2.0-dev libsndfile1-dev libmosquitto-dev \
        libpopt-dev libconfig-dev libasound2-dev avahi-daemon libavahi-client-dev \
        libssl-dev libsoxr-dev libplist-dev libsodium-dev libavutil-dev libavcodec-dev \
        libavformat-dev uuid-dev libgcrypt-dev xxd \
        libpulse-dev \
        alsa-utils \ -yy
}

function download_files () {
    local owner="$1"
    local pkgs="${@:2}"
    # Pre req
    # echo "cloning nqptp"
    # git clone https://github.com/mikebrady/nqptp.git
    # git clone https://github.com/mikebrady/shairport-sync.git
    for item in ${pkgs[@]}; do
        echo "cloning ${item}"
        download $owner $item
    done
}

function download () {
    local own="$1"
    local repo="$2"
    local tag="${3:-latest}"
    DEBUG echo "$own $repo"
    if [[ "${tag}" == "latest" ]]; then
        echo "attempting to get latest version for repo:$repo"
        tag=$(latest_tag $own $repo)
        DEBUG echo "$tag"
    fi
    # TODO: Add handlers for git clone error
    
    # --depth 1 --branch <tag_name>
    git clone --depth 1 --branch "$tag" "https://github.com/${own}/${repo}.git"
}


#? get the latest tagged version from github
function latest_tag () {
    local owner="$1"
    local repo="$2"
    curl -s "https://api.github.com/repos/${owner}/${repo}/releases/latest" | \
    grep "tag_name" | cut -d : -f 2,3 | tr -d \",
}


function install () {
    local pkg="$1"
    echo "installing $pkg" # arguments are accessible through $1, $2,...
    #? check if dir exists and cd
    [ -d "$pkg" ] && cd $pkg

    if (( $(id -u) == 0 )); then
        sudo make install
    else
        on_exit "$pkg failed to install correctly, please run with privilege"
    fi
}

function configure () {
    #? get all args
    local with_flags="$@"
    echo "$1" # arguments are accessible through $1, $2,...
    if [ -f "./configure" ]; then
        echo "File \"./configure\" exists"
        autoreconf -fi
        ./configure $with_flags # --with-systemd-startup
    fi
    DEBUG echo "./configure $with_flags" # --with-systemd-startup
}

function build () {
    make
}

function go_back() {
    cd ..
}

#? enable / start service
function service () {
    local service_cmd="$1"
    local service_name="$2"
    echo "$service_cmd service: $service_name" # arguments are accessible through $1, $2,...
    if systemctl is-active --quiet "$service_name.service"; then
        echo "$service_name running"
    else
        sudo systemctl $service_cmd "$service_name"
    fi
    # sudo systemctl enable nqptp
    # sudo systemctl start nqptp
}


#? Nqptp flags
# ./configure --with-systemd-startup

#? Shairport-Sync flags
# ./configure --sysconfdir=/etc --with-alsa \
#     --with-soxr --with-avahi --with-ssl=openssl \
#     --with-systemd --with-airplay-2 \
#     --with-mqtt-client --with-convolution \
#     --with-dbus-interface --with-mpris-interface \
#     # added by jason
#     --with-pa

remove_shairport() {
    rm $(which shairport-sync)
    rm /etc/systemd/system/shairport-sync.service \
    /etc/systemd/user/shairport-sync.service \
    /lib/systemd/system/shairport-sync.service \
    /lib/systemd/user/shairport-sync.service \
    /etc/init.d/shairport-sync
}

remove_nqptp() {
    rm /lib/systemd/system/nqptp.service \
    /usr/local/lib/systemd/system/nqptp.service
}

#? test shairport-sync
function test_shairportsync () {
    # get some diagnostics
    shairport-sync -v
    # get stats on audio recieved
    shairport-sync --statistics
}

#? arrays for options
#? section_array=(<with-flag>, <install_requirements>, <description>)
audio_output=('alsa::')
#? Audio Options
audio_options=( "soxr:libsoxr:Allows Shairport Sync to use libsoxr-based resampling for improved interpolation. Recommended."
        "apple-alac:libalac:Allows Shairport Sync to use the Apple ALAC Decoder. Requires libalac."
        "convolution:libsndfile:Includes a convolution filter that can be used to apply effects such as frequency and phase correction, and a loudness filter that compensates for the non-linearity of the human auditory system. Requires libsndfile."
    )

function option_help () {
    ARRAY=($1)
    # A pretend Python dictionary with bash 3 
    for flag in "${ARRAY[@]}" ; do
        KEY=${flag%%:*}
        VALUE=${flag#*:}
        REQ=${VALUE%%:*}
        VALUE=${VALUE#*:}
        echo "--with-${KEY}, requires: ${REQ} - ${VALUE}"
    done
}

option_help $audio_options 

#? Audio Output
# 'alsa' 'Output to the Advanced Linux Sound Architecture (ALSA) system. This is recommended for highest quality.'
# 'sndio' 'Output to the FreeBSD-native sndio system.'
# 'pa' 'Include the PulseAudio audio back end.'
# 'pw' 'Output to the PipeWire system.'
# 'ao' 'Output to the libao system. No synchronisation.'
# 'jack' 'Output to the Jack Audio system.'
# 'soundio' 'Include an optional backend module for audio to be output through the soundio system. No synchronisation.'
# 'stdout' 'Include an optional backend module to enable raw audio to be output through standard output (STDOUT).'
# 'pipe' 'Include an optional backend module to enable raw audio to be output through a unix pipe.'

# #? Metadata
# 'metadata' 'Adds support for Shairport Sync to request metadata and to pipe it to a compatible application. See https://github.com/mikebrady/shairport-sync-metadata-reader for a sample metadata reader.'

# #? IPC
# 'mqtt-client' 'Includes a client for MQTT,'
# 'dbus-interface' 'Includes support for the native Shairport Sync D-Bus interface'
# 'dbus-test-client' 'Compiles a D-Bus test client application'
# 'mpris-interface' 'Includes support for a D-Bus interface conforming as far as possible with the MPRIS standard,'
# 'mpris-test-client' 'Compiles an MPRIS test client application.'