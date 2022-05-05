#!/usr/bin/env sh
# Copyright 2022 ActiveState Software Inc. All rights reserved.

# URL to fetch update infos from.
BASE_INFO_URL="https://platform.activestate.com/sv/state-update/api/v1/info"
# URL to fetch installer archive from
BASE_FILE_URL="https://state-tool.s3.amazonaws.com/update/state"
# Path to the installer executable in the archive.
INSTALLERNAME="state-install/state-installer"
# Channel the installer will target
CHANNEL='release'
# The version to install (autodetermined to be the latest if left unspecified)
VERSION=""
# the download exetension
DOWNLOADEXT=".tar.gz"
# the installer extension
BINARYEXT=""
SHA256SUM="sha256sum"

SESSION_TOKEN_VERIFY="{TOKEN""}"
SESSION_TOKEN="{TOKEN}"
SESSION_TOKEN_VALUE=""

if [ "$SESSION_TOKEN" != "$SESSION_TOKEN_VERIFY" ]; then
  SESSION_TOKEN_VALUE=$SESSION_TOKEN
fi

getopt() {
  opt=$1; shift
  default=$1; shift
  i=0
  for arg in $@; do
    i=$((i + 1)) && [ "${arg}" != "$opt" ] && continue
    echo "$@" | cut -d' ' -f$(($i + 1)) && return
  done
  echo $default
}

CHANNEL=$(getopt "-b" "$CHANNEL" $@)
VERSION=$(getopt "-v" "$VERSION" $@)

if [ -z "${TERM}" ] || [ "${TERM}" = "dumb" ]; then
  OUTPUT_OK=""
  OUTPUT_ERROR=""
  OUTPUT_END=""
else
  OUTPUT_OK=`tput setaf 2`
  OUTPUT_ERROR=`tput setaf 1`
  OUTPUT_END=`tput sgr0`
fi

progress () {
  printf "• %s... " "$1"
}

progress_done() {
  echo "${OUTPUT_OK}✔ Done${OUTPUT_END}"
}

progress_fail() {
  echo "${OUTPUT_ERROR}x Failed${OUTPUT_END}"
}

error () {
  echo "$OUTPUT_ERROR${1}$OUTPUT_END"
}

# Determine the current OS.
case `uname -s` in
Linux)
  OS="linux"
  DOWNLOADEXT=".tar.gz"
  ;;
*BSD)
  OS=`uname -s | tr '[A-Z]' '[a-z]'`
  error "BSDs not supported yet"
  exit 1
  ;;
Darwin)
  OS="darwin"
  DOWNLOADEXT=".tar.gz"
  SHA256SUM="shasum -a 256"
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

if [ -z "$VERSION" ]; then
  # Determine the latest version to fetch.
  STATEURL="$BASE_INFO_URL?channel=$CHANNEL&source=install&platform=$OS"
  $FETCH $TMPDIR/info.json $STATEURL || exit 1

  # Parse info.
  VERSION=`cat $TMPDIR/info.json | sed -ne 's/.*"version":[ \t]*"\([^"]*\)".*/\1/p'`
  if [ -z "$VERSION" ]; then
    error "Unable to retrieve the latest version number"
    exit 1
  fi
  SUM=`cat $TMPDIR/info.json | sed -ne 's/.*"sha256":[ \t]*"\([^"]*\)".*/\1/p'`
  RELURL=`cat $TMPDIR/info.json | sed -ne 's/.*"path":[ \t]*"\([^"]*\)".*/\1/p'`
  rm $TMPDIR/info.json

else
  RELURL="$CHANNEL/$VERSION/$OS-amd64/state-$OS-amd64-$VERSION$DOWNLOADEXT"
fi

# Fetch the requested or latest version.
progress "Preparing Installer for State Tool Package Manager version $VERSION"
STATEURL="$BASE_FILE_URL/$RELURL"
ARCHIVE="$OS-amd64$DOWNLOADEXT"
$FETCH $TMPDIR/$ARCHIVE $STATEURL

# Verify checksum if possible.
if [ ! -z "$SUM" -a  "`$SHA256SUM -b $TMPDIR/$ARCHIVE | cut -d ' ' -f1`" != "$SUM" ]; then
  error "SHA256 sum did not match:"
  error "Expected: $SUM"
  error "Received: `$SHA256SUM -b $TMPDIR/$ARCHIVE | cut -d ' ' -f1`"
  error "Aborting installation."
  exit 1
fi

# Extract it.
if [ $OS = "windows" ]; then
  # Work around bug where MSYS produces a path that looks like `C:/temp` rather than `C:\temp`
  TMPDIRW=$(echo $(cd $TMPDIR && pwd -W) | sed 's|/|\\|g')
  powershell -command "& {&'Expand-Archive' -Force '$TMPDIRW\\$ARCHIVVE' '$TMPDIRW'}"
else
  tar -xzf $TMPDIR/$ARCHIVE -C $TMPDIR
fi
if [ $? -ne 0 ]; then
  progress_fail
  error "Could not download the State Tool installer at $STATEURL. Please try again."
  exit 1
fi

chmod +x $TMPDIR/$INSTALLERNAME$BINARYEXT
progress_done
echo ""

# Run the installer.
ACTIVESTATE_SESSION_TOKEN=$SESSION_TOKEN_VALUE $TMPDIR/$INSTALLERNAME$BINARYEXT "$@" --source-installer="install.sh"
