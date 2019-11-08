#!/usr/bin/env sh
# Copyright 2018 ActiveState Software Inc. All rights reserved.
#
# Usage: ./install.sh [-b branch]

echo "Installing the State Tool .."

USAGE=`cat <<EOF
install.sh [flags]

Flags:
 -b <branch>           Default 'unstable'.  Specify an alternative branch to install from (eg. master)
 -n                    Don't prompt for anything when installing into a new location
 -f                    Forces overwrite.  Overwrite existing state tool
 -t <dir>              Install into target directory <dir>
 -e <file>             Default 'state'. Filename to use for the executable
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
FORCEOVERWRITE=false

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
while getopts "nb:t:e:f?h-:" opt; do
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
    FORCEOVERWRITE=true
    ;;
  e)
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

# force overwrite requires no prompt flag
if $FORCEOVERWRITE && ( ! $NOPROMPT ); then
  error "Flag -f also requires -n"
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

# remove previous installation in temp dir
if [ -f $TMPDIR/$STATEPKG ]; then
  rm $TMPDIR/$STATEPKG
fi

if [ -f $TMPDIR/$TMPEXE ]; then
  rm $TMPDIR/$TMPEXE
fi

fetchArtifact () {
  info "Determining latest version..."
  # Determine the latest version to fetch.
  $FETCH $TMPDIR/$STATEJSON $STATEURL$STATEJSON || exit 1
  VERSION=`cat $TMPDIR/$STATEJSON | grep -m 1 '"Version":' | awk '{print $2}' | tr -d '",'`
  SUM=`cat $TMPDIR/$STATEJSON | grep -m 1 '"Sha256v2":' | awk '{print $2}' | tr -d '",'`
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

INSTALLDIR="`dirname \`which $STATEEXE\` 2>/dev/null`"

# stop if previous installation is detected unless
# - FORCEOVERWRITE is specified OR
# - a TARGET directory is specified that differs from INSTALLDIR
if [ ! -z "$INSTALLDIR" ] && ( ! $FORCEOVERWRITE ) && ( \
      [ -z $TARGET ] || [ $TARGET == $INSTALLDIR ] \
   ); then
  warn "Previous installation detected at $INSTALLDIR"
  echo "To update the state tool to the latest version, please run 'state update'."
  echo "To install in a different location, please specify the installation directory with '-t TARGET_DIR'."
  exit 0
fi

# Use target directory provided by user with no verification or default to
# one of two commonly used directories. 
# Ensure they are in PATH and if not use the first writable directory in PATH
if [ ! -z "$TARGET" ]; then
  INSTALLDIR=$TARGET
else
  if [ -w "/usr/local/bin" ]; then
    INSTALLDIR="/usr/local/bin"
  else
    INSTALLDIR="$HOME/.local/bin"
  fi
  # Verify the install directory is in PATH.
  INPATH=false
  OLDIFS=$IFS
  IFS=':'
  for PATHELEM in $PATH; do 
    if [ $INSTALLDIR = $PATHELEM ]; then
      INPATH=true
      break
    fi
  done

  # If the install directory is not in PATH we default to the first
  # directory in PATH that we have write access to as a last resort.
  if ! $INPATH; then
    for PATHELEM in $PATH; do
      if [ -w $PATHELEM ]; then
        INSTALLDIR=$PATHELEM
        break
      else
        INSTALLDIR=""
      fi
    done
  fi
  IFS=$OLDIFS
fi

if [ -z "$INSTALLDIR" ]; then
  error "Could not install state tool to PATH."
  error "You can use the '-t' flag to denote an install target."
  exit 1
fi

# Install to the determined directory.
while "true"; do
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
      fetchArtifact
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
  info "State tool installation complete."
  if [ -n "${ACTIVATE}" ]; then
    # switch this shell to interactive mode
    set -i
    # control flow of this script ends with this line: replace the shell with the activated project's shell
    exec $STATEEXE activate ${ACTIVATE}
  fi
  info "You may now start using the '$STATEEXE' program."
  exit 0
fi

manual_installation_instructions() {
  info "State tool installation complete."
  echo "Please manually add $INSTALLDIR to your \$PATH in order to start "
  echo "using the '$STATEEXE' program."
  echo "You can update your \$PATH by running 'export PATH=\$PATH:$INSTALLDIR'."
  echo "To make the changes to your path permanent please add the line"
  echo "'export PATH=\$PATH:$INSTALLDIR' to your $HOME/.profile file"
  activation_warning
  exit 1
}

# Prints a warning if an activation was requested and state tool is not in the PATH
activation_warning() {
  if [ -n "$ACTIVATE" ]; then
    echo
    warn "Cannot activate ${ACTIVATE} yet."
    echo "In order to activate a project, the state tool needs to be installed in your PATH first."
    echo "To manually activate the project run 'state activate ${ACTIVATE}' once 'state' is on your PATH"
  fi
}

# Check if we can write to the users profile, if not give manual
# insallation instructions
profile="`info $HOME`/.profile"
if [ ! -w "$profile" ]; then
  manual_installation_instructions
fi

# Prompt user to update users path, otherwise present manual
# installation instructions
userprompt "Allow \$PATH to be appended to in your $profile? [y/N]"
RESPONSE=$(userinput y | tr '[:upper:]' '[:lower:]')
if [ "$RESPONSE" != "y" ]; then
  manual_installation_instructions
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

info "State tool installation complete."
echo "Please either run 'source ~/.profile' or start a new login shell in "
echo "order to start using the '$STATEEXE' program."
activation_warning
exit 1
