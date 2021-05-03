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
 -e <file>                        Default 'state'. Filename to use for the executable
 -c <comand>                     Run any command after the install script has completed
 --activate <project>            Activate a project when State Tool is correctly installed
 --activate-default <project>    Activate a project and make it the system default
 -h                              Show usage information (what you're currently reading)
EOF
`

# ignore project file if we are already in an activated environment
unset ACTIVESTATE_PROJECT

# URL to fetch updates from.
STATEURL="https://state-tool.s3.amazonaws.com/update/state/release/"
# Name of the executable to ultimately use.
STATEEXE="state"
# Optional target directory
TARGET=""
# Optionally download and activate a project after install in the current directory
ACTIVATE=""
ACTIVATE_DEFAULT=""
POST_INSTALL_COMMAND=""

OS="linux"
SHA256SUM="sha256sum"
DOWNLOADEXT=".tar.gz"
BINARYEXT=""
ARCH="amd64"

NOPROMPT=false
FORCEOVERWRITE=false

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

# Determine the tmp directory.
if [ -z "$TMPDIR" ]; then
  TMPDIR="/tmp"
fi

# Process command line arguments.
while getopts "nb:t:e:c:f?h-:" opt; do
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
    STATEURL=`echo $STATEURL | sed -e "s|release|$OPTARG|;"`
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

if [ -n "$ACTIVATE" ] && [ -n "$ACTIVATE_DEFAULT" ]; then
  error "Flags --activate and --activate-default cannot be set at the same time."
  exit 1
fi

INSTALLDIR="`dirname \`which $STATEEXE\` 2>/dev/null`"

# stop if previous installation is detected unless
# - FORCEOVERWRITE is specified OR
# - a TARGET directory is specified that differs from INSTALLDIR
if [ ! -z "$INSTALLDIR" ] && ( ! $FORCEOVERWRITE ) && ( \
      [ -z $TARGET ] || [ "$TARGET" = "$INSTALLDIR" ] \
   ); then

  if [ -n "${ACTIVATE}" ]; then
    exec $INSTALLDIR/$STATEEXE activate ${ACTIVATE}
  elif [ -n "${ACTIVATE_DEFAULT}" ]; then
    exec $INSTALLDIR/$STATEEXE activate ${ACTIVATE_DEFAULT} --default
  fi

  warn "State Tool is already installed at $INSTALLDIR, to reinstall run this command again with -f"
  echo "To update the State Tool to the latest version, please run 'state update'."
  echo "To install in a different location, please specify the installation directory with '-t TARGET_DIR'."
  exit 0
fi

# If '-f' is passed and a previous installation exists we set NOPROMPT
# as we will overwrite the existing State Tool installation
if $FORCEOVERWRITE && [ ! -z "$INSTALLDIR" ]; then
  NOPROMPT=true
fi

echo "\
\033[2m╔═══════════════════════╗
║ \033[0m\033[39;1mInstalling State Tool\033[0m \033[2m║
╚═══════════════════════╝\033[0m"

CONSENT_TEXT="\

ActiveState collects usage statistics and diagnostic data about failures. The collected data complies with ActiveState Privacy Policy (https://www.activestate.com/company/privacy-policy/) and will be used to identify product enhancements, help fix defects, and prevent abuse.

By running the State Tool installer you consent to the Privacy Policy. This is required for the State Tool to operate while we are still in beta.
"
echo "$CONSENT_TEXT" | fold -s -w $WIDTH

# Construct system-dependent filenames.
STATEJSON=$OS-$ARCH.json
STATEPKG=$OS-$ARCH$DOWNLOADEXT
TMPEXE=$OS-$ARCH$BINARYEXT

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
  error "Could not install State Tool to PATH."
  error "You do not have write access to any directories currently on PATH."
  error "You can use the '-t' flag to denote an install target, "
  error "otherwise please ensure you have write permissions to a directory that's on your PATH."
  exit 1
fi

# Install to the determined directory.
info "Installing to $INSTALLDIR"
if [ ! -e "$INSTALLDIR" ]; then
  info "NOTE: $INSTALLDIR will be created"
elif [ -e "$INSTALLDIR/$STATEEXE" ]; then
  warn "WARNING: overwriting previous installation"
fi
if [ ! -z "`which $STATEEXE`" -a "`dirname \`which $STATEEXE\` 2>/dev/null`" != "$INSTALLDIR" ]; then
  warn "WARNING: installing elsewhere from previous installation"
fi
userprompt "Continue? [y/N] "
RESPONSE=$(userinput y)
case "$RESPONSE" in
  [Yy])
    # Install.
    if [ ! -e "$INSTALLDIR" ]; then
      mkdir -p "$INSTALLDIR" || continue
    fi
    fetchArtifact
    info "Installing to $INSTALLDIR..."
    mv $TMPDIR/$TMPEXE "$INSTALLDIR/$STATEEXE"
    ;;
  [Nn]|*)
    error "Aborting installation"
    exit 0
    ;;
esac


# If the installation is not in $PATH then we attempt to update the users rc file
if [ ! -z "$ZSH_VERSION" ] && [ -w "$HOME/.zshrc" ]; then
  info "Zsh shell detected"
  RC_FILE="$HOME/.zshrc"
elif [ ! -z "$BASH_VERSION" ] && [ -w "$HOME/.bashrc" ]; then
  info "Bash shell detected"
  RC_FILE="$HOME/.bashrc"
else
  RC_FILE="$HOME/.profile"
fi

manual_installation_instructions() {
  info "State Tool installation complete."
  echo "Please manually add $INSTALLDIR to your \$PATH in order to start "
  echo "using the '$STATEEXE' program."
  echo "You can update your \$PATH by running 'export PATH=\$PATH:$INSTALLDIR'."
  echo "To make the changes to your path permanent please add the line"
  echo "'export PATH=\$PATH:$INSTALLDIR' to your $RC_FILE file"
}

manual_update_instructions() {
  info "State Tool installation complete."
  # skip instruction to source rc file when we are activating
  if [ -n "${ACTIVATE}" ] || [ -n "${ACTIVATE_DEFAULT}" ]; then
    return
  fi
  echo "Please either run 'source $RC_FILE' or start a new login shell in "
  echo "order to start using the '$STATEEXE' program."
}

update_rc_file() {
  # Check if we can write to the users rcfile, if not give manual
  # insallation instructions
  if [ ! -w "$RC_FILE" ]; then
    warn "Could not write to $RC_FILE. Please ensure it exists and is writeable"
    manual_installation_instructions
  fi

  RC_KEY="# ActiveState State Tool"

  echo "Updating environment..."
  pathenv="export PATH=\"\$PATH:$INSTALLDIR\" $RC_KEY"
  if grep -q "$RC_KEY" $RC_FILE; then
    sed -i -E "s@^export.+$RC_KEY@$pathenv@" $RC_FILE
  else
    echo "" >> "$RC_FILE"
    echo "$pathenv" >> "$RC_FILE"
  fi
}

# Write install file
STATEPATH=$INSTALLDIR/$STATEEXE
CONFIGDIR=$($STATEPATH "export" "config" "--filter=dir")
echo "install.sh" > $CONFIGDIR/"installsource.txt"

$STATEPATH _prepare || exit $?

# Check if the installation is in $PATH, if so we also check if the activate
# flag was passed and attempt to activate the project
if [ "`dirname \`which $STATEEXE\` 2>/dev/null`" = "$INSTALLDIR" ]; then
  info "State Tool installation complete."
fi


if $NOPROMPT; then
  update_rc_file
  manual_update_instructions
else
  # Prompt user to update users path, otherwise present manual
  # installation instructions
  userprompt "Allow \$PATH to be appended in your $RC_FILE? [y/N]"
  RESPONSE=$(userinput y | tr '[:upper:]' '[:lower:]')
  if [ "$RESPONSE" != "y" ]; then
    manual_installation_instructions
  else
    update_rc_file
    manual_update_instructions
  fi
fi

# Keep --activate and --activate-default flags for backwards compatibility
if [ -n "${POST_INSTALL_COMMAND}" ]; then
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
