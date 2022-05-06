#!/usr/bin/env sh
# Copyright 2022 ActiveState Software Inc. All rights reserved.

# URL to fetch update infos from.
BASE_INFO_URL="https://platform.activestate.com/sv/state-update/api/v1/info"
# URL to fetch installer archive from
BASE_FILE_URL="https://state-tool.s3.amazonaws.com/update/state"
# Path to the installer executable in the archive.
INSTALLERNAME="state-install/state-installer"
# Channel the installer will target
CHANNEL="release"
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
# wget and curl differ on how to handle AWS' "Forbidden" result for unknown versions.
# wget will exit with nonzero status. curl simply creates an XML file with the forbidden error.
# If curl was used, make sure the file downloaded is of type 'data', according to the UNIX `file`
# command. (The XML error will be reported as a 'text' type.)
# If wget returned an error or curl fetched a "forbidden" response, raise an error and exit.
if [ $? -ne 0 -o \( "`echo $FETCH | grep -o 'curl'`" == "curl" -a -z "`file -b $TMPDIR/$ARCHIVE | grep -o 'data'`" \) ]; then
  rm -f $TMPDIR/$ARCHIVE
  progress_fail
  error "Could not download the State Tool installer at $STATEURL. Please try again."
  exit 1
fi

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
  powershell -command "& {&'Expand-Archive' -Force '$TMPDIRW\\$ARCHIVE' '$TMPDIRW'}"
else
  tar -xzf $TMPDIR/$ARCHIVE -C $TMPDIR || exit 1
fi
chmod +x $TMPDIR/$INSTALLERNAME$BINARYEXT
progress_done
echo ""

# Run the installer.
ACTIVESTATE_SESSION_TOKEN=$SESSION_TOKEN_VALUE $TMPDIR/$INSTALLERNAME$BINARYEXT "$@" --channel="$CHANNEL" --source-installer="install.sh"
