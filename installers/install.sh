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
  ARCH="amd64"
  arch="`uname -m`"
  if [ $arch = "arm64" -o $arch = "aarch64"  ]; then ARCH="arm64"; fi
  DOWNLOADEXT=".tar.gz"
  ;;
*BSD)
  OS=`uname -s | tr '[A-Z]' '[a-z]'`
  error "BSDs not supported yet"
  exit 1
  ;;
Darwin)
  OS="darwin"
  ARCH="amd64"
  DOWNLOADEXT=".tar.gz"
  SHA256SUM="shasum -a 256"
  ;;
MINGW*|MSYS*)
  OS="windows"
  ARCH="amd64"
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

INSTALLERTMPDIR="$TMPDIR/state-install-$RANDOM"
mkdir -p "$INSTALLERTMPDIR"

if [ -z "$VERSION" ]; then
  # If the user did not specify a version, formulate a query to fetch the JSON info of the latest
  # version, including where it is.
  JSONURL="$BASE_INFO_URL?channel=$CHANNEL&source=install&platform=$OS&arch=$ARCH"
elif [ -z "`echo $VERSION | grep -o '\-SHA'`" ]; then
  # If the user specified a partial version (i.e. no SHA), formulate a query to fetch the JSON info
  # of that version's latest SHA, including where it is.
  VERSIONNOSHA="$VERSION"
  VERSION=""
  JSONURL="$BASE_INFO_URL?channel=$CHANNEL&source=install&platform=$OS&arch=$ARCH&target-version=$VERSIONNOSHA"
else
  # If the user specified a full version with SHA, formulate a query to fetch the JSON info of that
  # version.
  VERSIONNOSHA="`echo $VERSION | sed 's/-SHA.*$//'`"
  JSONURL="$BASE_INFO_URL?channel=$CHANNEL&source=install&platform=$OS&arch=$ARCH&target-version=$VERSIONNOSHA"
fi

# Fetch version info.
$FETCH $INSTALLERTMPDIR/info.json $JSONURL
if [ $? -ne 0 -o ! -z "`grep -o Invalid $INSTALLERTMPDIR/info.json`" ]; then
  error_message=$(grep -o '"message":"[^"]*"' $INSTALLERTMPDIR/info.json | cut -d':' -f2 | sed 's/"//g')
  if [ -n "$error_message" ]; then
    error "Could not download a State Tool installer, recieved error message: $error_message"
  else
      error "Could not download a State Tool installer for the given command line arguments"
  fi
	exit 1
fi

# Extract checksum.
SUM=`cat $INSTALLERTMPDIR/info.json | sed -ne 's/.*"sha256":[ \t]*"\([^"]*\)".*/\1/p'`

if [ -z "$VERSION" ]; then
  # If the user specified no version or a partial version we need to use the json URL to get the
  # actual installer URL.
  VERSION=`cat $INSTALLERTMPDIR/info.json | sed -ne 's/.*"version":[ \t]*"\([^"]*\)".*/\1/p'`
  if [ -z "$VERSION" ]; then
    error "Unable to retrieve the latest version number"
    exit 1
  fi
  RELURL=`cat $INSTALLERTMPDIR/info.json | sed -ne 's/.*"path":[ \t]*"\([^"]*\)".*/\1/p'`
else
  # If the user specified a full version, construct the installer URL.
  if [ "$VERSION" != "`cat $INSTALLERTMPDIR/info.json | sed -ne 's/.*"version":[ \t]*"\([^"]*\)".*/\1/p'`" ]; then
    error "Unknown version: $VERSION"
    exit 1
  fi
  RELURL="$CHANNEL/$VERSIONNOSHA/$OS-$ARCH/state-$OS-$ARCH-$VERSION$DOWNLOADEXT"
fi

# Fetch the requested or latest version.
progress "Preparing Installer for State Tool Package Manager version $VERSION"
STATEURL="$BASE_FILE_URL/$RELURL"
ARCHIVE="$OS-$ARCH$DOWNLOADEXT"
$FETCH $INSTALLERTMPDIR/$ARCHIVE $STATEURL
# wget and curl differ on how to handle AWS' "Forbidden" result for unknown versions.
# wget will exit with nonzero status. curl simply creates an XML file with the forbidden error.
# If curl was used, make sure the file downloaded is not an XML file (i.e. it does not start with "<?xml").
# If wget returned an error or curl fetched a "forbidden" response, raise an error and exit.
if [ $? -ne 0 -o \( "`echo $FETCH | grep -o 'curl'`" = "curl" -a ! -z "`grep -o '^<?xml' $INSTALLERTMPDIR/$ARCHIVE`" \) ]; then
  rm -f $INSTALLERTMPDIR/$ARCHIVE
  progress_fail
  error "Could not download the State Tool installer at $STATEURL. Please try again."
  exit 1
fi

# Verify checksum.
if [ "`$SHA256SUM -b $INSTALLERTMPDIR/$ARCHIVE | cut -d ' ' -f1`" != "$SUM" ]; then
  error "SHA256 sum did not match:"
  error "Expected: $SUM"
  error "Received: `$SHA256SUM -b $INSTALLERTMPDIR/$ARCHIVE | cut -d ' ' -f1`"
  error "Aborting installation."
  exit 1
fi

# Extract it.
if [ $OS = "windows" ]; then
  # Work around bug where MSYS produces a path that looks like `C:/temp` rather than `C:\temp`
  INSTALLERTMPDIRW=$(echo $(cd $INSTALLERTMPDIR && pwd -W) | sed 's|/|\\|g')
  powershell -command "& {&'Expand-Archive' -Force '$INSTALLERTMPDIRW\\$ARCHIVE' '$INSTALLERTMPDIRW'}"
else
  tar -xzf $INSTALLERTMPDIR/$ARCHIVE -C $INSTALLERTMPDIR || exit 1
fi
chmod +x $INSTALLERTMPDIR/$INSTALLERNAME$BINARYEXT
progress_done
echo ""

# Run the installer.
ACTIVESTATE_SESSION_TOKEN=$SESSION_TOKEN_VALUE $INSTALLERTMPDIR/$INSTALLERNAME$BINARYEXT "$@" --source-installer="install.sh"

# Remove temp files
rm -r $INSTALLERTMPDIR
