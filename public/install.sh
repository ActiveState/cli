#!/usr/bin/env sh
# Copyright 2018 ActiveState Software Inc. All rights reserved.
#
# Usage: ./install.sh [-b branch]

echo "Installing the State Tool .."

USAGE=`cat <<EOF
install.sh [flags]

Flags:
 -b <branch>           Default 'unstable'.  Specify an alternative branch to install from (eg. master)
 -n                    Don't prompt for anything, just install and override any existing executables
 -t <dir>              Install target directory
 -f <file>             Default 'state'.  Binary filename to use
 --activate <project>  Activate a project when state tools is correctly installed
 -h                    Show usage information (what you're currently reading)
EOF
`

# URL to fetch updates from.
STATEURL="https://s3.ca-central-1.amazonaws.com/cli-update/update/state/unstable/"
# Name of the executable to ultimately use.
STATEEXE="state"
# ID of the $PATH entry in the user's ~/.profile for the executable.
STATEID="ActiveStateCLI"
# Optional target directory
TARGET=""
# Optionally download and activate a project after install in the current directory
ACTIVATE=""

OS="linux"
SHA256SUM="sha256sum"
DOWNLOADEXT=".tar.gz"
BINARYEXT=""
ARCH="amd64"

TERM="${TERM:=xterm}"

NOPROMPT=false

info () {
  echo "$(tput bold)==> ${1}$(tput sgr0)"
}

warn () {
  echo "$(tput setf 3)${1}$(tput sgr0)"
}

error () {
  echo "$(tput setf 1)${1}$(tput sgr0)"
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

# Determine the tmp directory.
if [ -z "$TMPDIR" ]; then
  TMPDIR="/tmp"
fi

# Process command line arguments.
while getopts "nb:t:f:?h-:" opt; do
  case $opt in
  -)  # parse long options
    case ${OPTARG} in
      activate)
        # zsh compliant indirection, gathering the next command line argument
        eval "ACTIVATE=\"\${${OPTIND}}\""
        OPTIND=$(( OPTIND + 1 ))
        ;;
    esac
    ;;
  b)
    STATEURL=`echo $STATEURL | sed -e "s/unstable/$OPTARG/;"`
    ;;
  t)
    TARGET=$OPTARG
    ;;
  f)
    STATEEXE=$OPTARG
    ;;
  n)
    NOPROMPT=true
    ;;
  h|?)
    echo "${USAGE}"
    exit 0
    ;;
  esac
done

# state activate currently does not run without user interaction, 
# so we are bailing if that's being requested...
if $NOPROMPT && [ -n "$ACTIVATE" ]; then
  error "Flags -n and --activate cannot be set at the same time."
  exit 1
fi

# Construct system-dependent filenames.
STATEJSON=$OS-$ARCH.json
STATEPKG=$OS-$ARCH$DOWNLOADEXT
TMPEXE=$OS-$ARCH$BINARYEXT

info "${PREFIX}Preparing for installation...${SUFFIX}"

# Determine a fetch method
if [ ! -z "`command -v wget`" ]; then
  FETCH="wget -nv -O"
elif [ ! -z "`command -v curl`" ]; then
  FETCH="curl -sS -o"
else
  error "Either wget or curl is required to download files"
  exit 1
fi

if [ -f $TMPDIR/$STATEPKG ]; then
  rm $TMPDIR/$STATEPKG
fi

if [ -f $TMPDIR/$TMPEXE ]; then
  rm $TMPDIR/$TMPEXE
fi

info "Determining latest version..."
# Determine the latest version to fetch.
$FETCH $TMPDIR/$STATEJSON $STATEURL$STATEJSON || exit 1
VERSION=`cat $TMPDIR/$STATEJSON | grep -m 1 '"Version":' | awk '{print $2}' | tr -d '",'`
rm $TMPDIR/$STATEJSON
if [ -z "$VERSION" ]; then
  error "Unable to retrieve the latest version number"
  exit 1
fi
info "Fetching the latest version: $VERSION..."
# Fetch it.
$FETCH $TMPDIR/$STATEPKG ${STATEURL}${VERSION}/${STATEPKG} || exit 1

# Extract the State binary after verifying its checksum.
# Verify checksum.
info "Verifying checksum..."
SUM=`$FETCH - $STATEURL$STATEJSON | grep -m 1 '"Sha256v2":' | awk '{print $2}' | tr -d '",'`
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

# Check for existing installation. Otherwise, make the installation default to
# /usr/local/bin if the user has write permission, or to a local bin.
INSTALLDIR="`dirname \`which $STATEEXE\` 2>/dev/null`"
if [ ! -z "$TARGET" ]; then
  INSTALLDIR=$TARGET
elif [ ! -z "$INSTALLDIR" ]; then
  warn "Previous installation detected at $INSTALLDIR"
else
  if [ -w "/usr/local/bin" ]; then
    INSTALLDIR="/usr/local/bin"
  else
    INSTALLDIR="$HOME/.local/bin"
  fi
fi

# Prompt the user for a directory to install to.
while "true"; do
  userprompt "Please enter the installation directory [$INSTALLDIR]: "
  INPUT=$(userinput $INSTALLDIR)
  if [ -e "$INPUT" -a ! -d "$INPUT" ]; then
    warn "$INPUT exists and is not a directory"
    continue
  elif [ -e "$INPUT" -a ! -w "$INPUT" ]; then
    warn "You do not have permission to write to $INPUT"
    continue
  fi
  if [ ! -z "$INPUT" ]; then
    if [ ! -z "`realpath \"$INPUT\" 2>/dev/null`" ]; then
      INSTALLDIR="`realpath \"$INPUT\"`"
    else
      INSTALLDIR="$INPUT"
    fi
  fi
  info "Installing to $INSTALLDIR"
  if [ ! -e "$INSTALLDIR" ]; then
    info "NOTE: $INSTALLDIR will be created"
  elif [ -e "$INSTALLDIR/$STATEEXE" ]; then
    warn "WARNING: overwriting previous installation"
  fi
  if [ ! -z "`which $STATEEXE`" -a "`dirname \`which $STATEEXE\` 2>/dev/null`" != "$INSTALLDIR" ]; then
    warn "WARNING: installing elsewhere from previous installation"
  fi
  userprompt "Continue? [y/N/q] "
  RESPONSE=$(userinput y)
  case "$RESPONSE" in
    [Qq])
      error "Aborting installation"
      exit 0
      ;;
    [Yy])
      # Install.
      if [ ! -e "$INSTALLDIR" ]; then
        mkdir -p "$INSTALLDIR" || continue
      fi
      info "Installing to $INSTALLDIR..."
      mv $TMPDIR/$TMPEXE "$INSTALLDIR/$STATEEXE"
      if [ $? -eq 0 ]; then
        break
      fi
      ;;
    [Nn]|*)
      continue
      ;;
  esac
done

# Check if the installation is in $PATH, if not, update user's profile if
# permitted to.
if [ "`dirname \`which $STATEEXE\` 2>/dev/null`" = "$INSTALLDIR" ]; then
  info "Installation complete."
  if [ -n "${ACTIVATE}" ]; then
    # switch this shell to interactive mode
    set -i
    # control flow of this script ends with this line: replace the shell with the activated project's shell
    exec $STATEEXE activate ${ACTIVATE}
  fi
  info "You may now start using the '$STATEEXE' program."
  exit 0
fi

# Beyond this point, the state tool is not in the PATH and therefor unsafe to execute.

# Prints a warning if an activation was requested and state tool is not in the PATH
activation_warning() {
  if [ -n "$ACTIVATE" ]; then
    echo
    warn "Cannot activate ${ACTIVATE} yet."
    echo "In order to activate a project, the state tool needs to be installed in your PATH first."
    echo "To manually activate the project run 'state activate ${ACTIVATE}' once 'state' is on your PATH"
  fi
}

profile="`info $HOME`/.profile"
if [ ! -w "$profile" ]; then
  info "Installation complete."
  echo "Please manually add $INSTALLDIR to your \$PATH in order to start "
  echo "using the '$STATEEXE' program."
  activation_warning
  exit 0
fi

userprompt "Allow \$PATH to be appended to in your $profile? [y/N]"
RESPONSE=$(userinput y | tr '[:upper:]' '[:lower:]')
if [ "$RESPONSE" != "y" ]; then
  info "Installation complete."
  echo "Please manually add $INSTALLDIR to your \$PATH in order to start "
  echo "using the '$STATEEXE' program."
  activation_warning
  exit 0
fi
info "Updating environment..."
pathenv="export PATH=\"\$PATH:$INSTALLDIR\" #$STATEID"
if [ -z "`grep -no \"\#$STATEID\" \"$profile\"`" ]; then
  info "Adding to \$PATH in $profile"
  info "\n$pathenv" >> "$profile"
else
  info "Updating \$PATH in $profile"
  sed -i -e "s|^export PATH=[^\#]\+\#$STATEID|$pathenv|;" "$profile"
fi

info "Installation complete."
echo "Please either run 'source ~/.profile' or start a new login shell in "
echo "order to start using the '$STATEEXE' program."

activation_warning
