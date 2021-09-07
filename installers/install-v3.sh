#!/usr/bin/env sh
# Copyright 2021 ActiveState Software Inc. All rights reserved.

# URL to fetch update files from
BASE_FILE_URL="https://state-tool.s3.amazonaws.com/update/state"
# Name of the executable to ultimately use.
INSTALLEREXE="state-installer"
# channel the installer will target
CHANNEL='release'
# the name of the remote archive to download
INSTALLERNAME="state-installer"
# the download exetension
DOWNLOADEXT=".tar.gz"
# the installer extension
BINARYEXT=""

if [ -z "${TERM}" ] || [ "${TERM}" = "dumb" ]; then
  OUTPUT_BOLD=""
  OUTPUT_WARN=""
  OUTPUT_ERROR=""
  OUTPUT_END=""
  WIDTH=80
else
  OUTPUT_BOLD="$(tput bold)"
  OUTPUT_WARN="$(tput setf 3)"
  OUTPUT_ERROR="$(tput setf 1)"
  OUTPUT_END="$(tput sgr0)"
  WIDTH="$(tput cols)"
fi

info () {
  echo "░▒▓█$OUTPUT_BOLD${1}$OUTPUT_END"
}

warn () {
  echo "$OUTPUT_WARN${1}$OUTPUT_END"
}

error () {
  echo "$OUTPUT_ERROR${1}$OUTPUT_END"
}

# Determine the current OS.
case `uname -s` in
Linux)
  # Defaults already cover Linux
  ;;
*BSD)
  OS=`uname -s | tr '[A-Z]' '[a-z]'`
  error "BSDs not supported yet"
  exit 1
  ;;
Darwin)
  OS="darwin"
  ;;
MINGW*|MSYS*)
  OS="windows"
  DOWNLOADEXT=".zip"
  BINARYEXT=".exe"
  ;;
*)
  error "Unsupported OS: `uname -s`"
  exit 1
  ;;
esac

# Determine a fetch method
if [ ! -z "`command -v wget`" ]; then
  FETCH="wget -q -O"
elif [ ! -z "`command -v curl`" ]; then
  FETCH="curl -sS -o"
else
  error "Either wget or curl is required to download files"
  exit 1
fi

# Determine the tmp directory.
if [ -z "$TMPDIR" ]; then
  TMPDIR="/tmp"
fi

info "Preparing ActiveState Installer"

STATEURL="$STATEURL/$CHANNEL/$INSTALLERNAME$DOWNLOADEXT"
ARCHIVE="$TMPDIR/$INSTALLERNAME$DOWNLOADEXT"
INSTALLER="$TMDIR/$INSTALLERNAME$BINARYEXT"
if ! $FETCH $ARCHIVE $STATEURL ; then
  error "Could not fetch version: $VERSION, please verify the version number and try again."
  exit 1
fi

info "Extracting $ARCHIVE..."
if [ $OS = "windows" ]; then
  # Work around bug where MSYS produces a path that looks like `C:/temp` rather than `C:\temp`
  TMPDIRW=$(echo $(cd $TMPDIR && pwd -W) | sed 's|/|\\|g')
  powershell -command "& {&'Expand-Archive' -Force '$TMPDIRW\\$ARCHIVVE' '$TMPDIRW'}"
else
  tar -xzf $TMPDIR/$ARCHIVE -C $TMPDIR || exit 1
fi
chmod +x $TMPDIR/$INSTALLER

$TMPDIR/$INSTALLER "$@"