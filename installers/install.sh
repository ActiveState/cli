#!/usr/bin/env sh
# Copyright 2018 ActiveState Software Inc. All rights reserved.
#
# Usage: ./install.sh [-b branch]

echo "Installing the State Tool .."

USAGE=`cat <<EOF
install.sh [flags]

Flags:
 -b <branch>                     Default 'release'.  Specify an alternative branch to install from (eg. beta)
 -n                              Don't prompt for anything when installing into a new location
 -f                              Forces overwrite.  Overwrite existing State Tool
 -t <dir>                        Install into target directory <dir>
 -c <comand>                     Run any command after the install script has completed
 --activate <project>            Activate a project when State Tool is correctly installed
 --activate-default <project>    Activate a project and make it the system default
 -h                              Show usage information (what you're currently reading)
 -v <version-SHA>                The version of the State Tool to install
EOF
`

# ignore project file if we are already in an activated environment
unset ACTIVESTATE_PROJECT

# URL to fetch update infos from.
BASE_INFO_URL="https://<to-be-determined>/info"
# URL to fetch update files from
BASE_FILE_URL="https://state-tool.s3.amazonaws.com/update/state"
# Name of the executable to ultimately use.
STATEEXE="state"
# Optional target directory
TARGET=""
# Optionally download and activate a project after install in the current directory
ACTIVATE=""
ACTIVATE_DEFAULT=""
POST_INSTALL_COMMAND=""
VERSION=""

OS="linux"
SHA256SUM="sha256sum"
DOWNLOADEXT=".tar.gz"
BINARYEXT=""
ARCH="amd64"

NOPROMPT=false
FORCEOVERWRITE=false

SESSION_TOKEN_VERIFY="{TOKEN""}"
SESSION_TOKEN="{TOKEN}"
SESSION_TOKEN_VALUE=""

if [ "$SESSION_TOKEN" != "$SESSION_TOKEN_VERIFY" ]; then
  SESSION_TOKEN_VALUE=$SESSION_TOKEN
fi

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
  echo "$OUTPUT_BOLD==> ${1}$OUTPUT_END"
}

warn () {
  echo "$OUTPUT_WARN${1}$OUTPUT_END"
}

error () {
  echo "$OUTPUT_ERROR${1}$OUTPUT_END"
}

userprompt () {
  if ! $NOPROMPT ; then
    echo "$1"
  fi
}

userinput () {
  if $NOPROMPT ; then
    echo "$1"
  else 
    read result
    echo "$result"
  fi
}


# Determine the current OS.
case `uname -s` in
Linux)
  # Defaults already cover Linux
  ;;
*BSD)
  OS=`uname -s | tr '[A-Z]' '[a-z]'`
  SHA256SUM=""
  error "BSDs not supported yet"
  exit 1
  ;;
Darwin)
  OS="darwin"
  SHA256SUM="shasum -a 256"
  ;;
MINGW*|MSYS*)
  OS="windows"
  DOWNLOADEXT=".zip"
  BINARYEXT=".exe"
  STATEEXE=${STATEEXE}.exe
  ;;
*)
  error "Unsupported OS: `uname -s`"
  exit 1
  ;;
esac

# Determine the current architecture.
case `uname -m` in
i?86)
  ARCH="386"
  ;;
x86_64)
  # Covered by default value
  ;;
esac

set_tempdir () {
  if type mktemp > /dev/null; then
    TMPDIR=`mktemp -d`
  else
    TMPDIR="${TMPDIR:-/tmp}/state-installer.$$"
    # clean-up previous temp dir
    rm -rf $tdir
    mkdir -p $TMPDIR
  fi
}

set_tempdir

CHANNEL='release'
# Process command line arguments.
while getopts "nb:t:e:c:v:f?h-:" opt; do
  case $opt in
  -)  # parse long options
    case ${OPTARG} in
      activate)
        # zsh compliant indirection, gathering the next command line argument
        eval "ACTIVATE=\"\${${OPTIND}}\""
        OPTIND=$(( OPTIND + 1 ))
        ;;
      activate-default)
        eval "ACTIVATE_DEFAULT=\"\${${OPTIND}}\""
        OPTIND=$(( OPTIND + 1 ))
        ;;
    esac
    ;;
  b)
    CHANNEL=$OPTARG
    ;;
  c)
    POST_INSTALL_COMMAND=$OPTARG
    ;;
  t)
    TARGET=$OPTARG
    ;;
  f)
    FORCEOVERWRITE=true
    ;;
  n)
    NOPROMPT=true
    ;;
  v)
    VERSION=$OPTARG
    ;;
  h|?)
    echo "${USAGE}"
    exit 0
    ;;
  esac
done

STATEURL="$BASE_INFO_URL?channel=$CHANNEL&source=install&platform=$OS"

# state activate currently does not run without user interaction, 
# so we are bailing if that's being requested...
if $NOPROMPT && [ -n "$ACTIVATE" ]; then
  error "Flags -n and --activate cannot be set at the same time."
  exit 1
fi

if [ -n "$ACTIVATE" ] && [ -n "$ACTIVATE_DEFAULT" ]; then
  error "Flags --activate and --activate-default cannot be set at the same time."
  exit 1
fi

CURRENT_INSTALLDIR="`dirname \`which $STATEEXE\` 2>/dev/null`"

# stop if previous installation is detected unless
# - FORCEOVERWRITE is specified OR
# - a TARGET directory is specified that differs from CURRENT_INSTALLDIR
if [ ! -z "$CURRENT_INSTALLDIR" ] && ( ! $FORCEOVERWRITE ) && ( \
      [ -z $TARGET ] || [ $TARGET -ef $CURRENT_INSTALLDIR ] \
   ); then

  if [ -n "${ACTIVATE}" ]; then
    exec $CURRENT_INSTALLDIR/$STATEEXE activate ${ACTIVATE}
  elif [ -n "${ACTIVATE_DEFAULT}" ]; then
    exec $CURRENT_INSTALLDIR/$STATEEXE activate ${ACTIVATE_DEFAULT} --default
  fi

  warn "State Tool is already installed at $CURRENT_INSTALLDIR, to reinstall run this command again with -f"
  echo "To update the State Tool to the latest version, please run 'state update'."
  echo "To install in a different location, please specify the installation directory with '-t TARGET_DIR'."
  exit 0
fi

# If '-f' is passed and a previous installation exists we set NOPROMPT
# as we will overwrite the existing State Tool installation
if $FORCEOVERWRITE && [ ! -z "$CURRENT_INSTALLDIR" ]; then
  NOPROMPT=true
fi

echo "\
\033[2m╔═══════════════════════╗
║ \033[0m\033[39;1mInstalling State Tool\033[0m \033[2m║
╚═══════════════════════╝\033[0m"

CONSENT_TEXT="\

ActiveState collects usage statistics and diagnostic data about failures. The collected data complies with ActiveState Privacy Policy (https://www.activestate.com/company/privacy-policy/) and will be used to identify product enhancements, help fix defects, and prevent abuse.

By running the State Tool installer you consent to the Privacy Policy. This is required for the State Tool to operate while we are still in beta.

Please note that the installer may modify your shell configuration file (eg., .bashrc) to add the installation PATH to your environment.
"
echo "$CONSENT_TEXT" | fold -s -w $WIDTH

# Construct system-dependent filenames.
STATEPKG=$OS-$ARCH$DOWNLOADEXT
TMPEXE="state-install/state-installer"$BINARYEXT

info "${PREFIX}Preparing for installation...${SUFFIX}"

# Determine a fetch method
if [ ! -z "`command -v wget`" ]; then
  FETCH="wget -q -O"
elif [ ! -z "`command -v curl`" ]; then
  FETCH="curl -sS -o"
else
  error "Either wget or curl is required to download files"
  exit 1
fi

fetchArtifact () {
  if [ ! -z "$VERSION" ]; then
    info "Attempting to fetch version: $VERSION..."
    STATEURL="$STATEURL&target-version=$VERSION"
    if ! $FETCH $TMPDIR/info.json $STATEURL ; then
      error "Could not fetch version: $VERSION, please verify the version number and try again."
      exit 1
    fi

    info "Fetching version: $VERSION..."
  else
    info "Determining latest version..."
    # Determine the latest version to fetch.
    $FETCH $TMPDIR/info.json $STATEURL || exit 1
    VERSION=`cat $TMPDIR/info.json | grep -m 1 '"version":' | awk '{print $2}' | tr -d '",'`

    if [ -z "$VERSION" ]; then
      error "Unable to retrieve the latest version number"
      exit 1
    fi

    info "Fetching the latest version: $VERSION..."
  fi

  UPDATE_TAG=`cat $TMPDIR/info.json | grep -m 1 '"tag":' | awk '{print $2}' | tr -d '",}'`
  SUM=`cat $TMPDIR/info.json | grep -m 1 '"sha256":' | awk '{print $2}' | tr -d '",'`
  RELURL=`cat $TMPDIR/info.json | grep -m 1 '"path":' | awk '{print $2}' | tr -d '",'`
  rm $TMPDIR/info.json


  URL="${BASE_FILE_URL}/${RELURL}"
  # Fetch it.
  $FETCH $TMPDIR/$STATEPKG ${URL} || exit 1

  # Extract the State binary after verifying its checksum.
  # Verify checksum.
  info "Verifying checksum..."
  if [ "`$SHA256SUM -b $TMPDIR/$STATEPKG | cut -d ' ' -f1`" != "$SUM" ]; then
    error "SHA256 sum did not match:"
    error "Expected: $SUM"
    error "Received: `$SHA256SUM -b $TMPDIR/$STATEPKG | cut -d ' ' -f1`"
    error "Aborting installation."
    exit 1
  fi

  info "Extracting $STATEPKG..."
  if [ $OS = "windows" ]; then
    # Work around bug where MSYS produces a path that looks like `C:/temp` rather than `C:\temp`
    TMPDIRW=$(echo $(cd $TMPDIR && pwd -W) | sed 's|/|\\|g')
    powershell -command "& {&'Expand-Archive' -Force '$TMPDIRW\\$STATEPKG' '$TMPDIRW'}"
  else
    tar -xzf $TMPDIR/$STATEPKG -C $TMPDIR || exit 1
  fi
  chmod +x $TMPDIR/$TMPEXE
}

if [ ! -z "`which $STATEEXE`" -a "`dirname \`which $STATEEXE\` 2>/dev/null`" != "$CURRENT_INSTALLDIR" ]; then
  warn "WARNING: installing elsewhere from previous installation"
fi
userprompt "Accept terms and proceed with install? [Y/n] "
RESPONSE=$(userinput y)
case "$RESPONSE" in
  [Nn])
    error "Aborting installation"
    exit 0
    ;;
  [Yy]|*)
    fetchArtifact
    OUTPUT_FILE=$TMPDIR/install_output.txt
    if [ ! -z "$TARGET" ]; then
      ACTIVESTATE_UPDATE_TAG=$UPDATE_TAG ACTIVESTATE_SESSION_TOKEN=$SESSION_TOKEN_VALUE "$TMPDIR/$TMPEXE" "$TARGET" 2>&1 | tee $OUTPUT_FILE
    else
      ACTIVESTATE_UPDATE_TAG=$UPDATE_TAG ACTIVESTATE_SESSION_TOKEN=$SESSION_TOKEN_VALUE "$TMPDIR/$TMPEXE" 2>&1 | tee $OUTPUT_FILE
    fi
    INSTALL_OUTPUT=$(cat $OUTPUT_FILE)
    rm -f $OUTPUT_FILE
    INSTALLDIR=$(echo $INSTALL_OUTPUT | sed -n 's/.*Install Location: //p' | cut -f1 -d" ")
    ;;
esac

# Write install file
STATEPATH=$INSTALLDIR/$STATEEXE
CONFIGDIR=$($STATEPATH "--output=simple" "export" "config" "--filter=dir")
echo "install.sh" > $CONFIGDIR/"installsource.txt"

info "State Tool successfully installed."
info "Reminder: Start a new shell in order to start using the State Tool."

# Keep --activate and --activate-default flags for backwards compatibility
if [ -n "${POST_INSTALL_COMMAND}" ]; then
  # Ensure that new installation dir is on the PATH for follow up commands
  export PATH="$PATH:$INSTALLDIR"
  exec $POST_INSTALL_COMMAND
elif [ -n "${ACTIVATE}" ]; then
  # control flow of this script ends with this line: replace the shell with the activated project's shell
  exec $STATEPATH activate ${ACTIVATE}
elif [ -n "${ACTIVATE_DEFAULT}" ]; then
  exec $STATEPATH activate ${ACTIVATE_DEFAULT} --default
else
  echo "\n\
\033[32m╔══════════════════════╗
║ \033[0m\033[39;1mState Tool Installed\033[0m \033[32m║
╚══════════════════════╝\033[0m"
fi
