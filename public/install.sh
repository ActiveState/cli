#!/bin/sh
# Copyright 2018 ActiveState Software Inc. All rights reserved.
#
# Usage: ./install.sh [-b branch]

USAGE=`cat <<EOF
install.sh [flags]

Flags:
 -b <branch>      Specify an alternative branch to install from (eg. master)
 -n               Don't prompt for anything, just install and override any existing executables
 -h               Shows usage information (what you're currently reading)
EOF
`

# URL to fetch updates from.
STATEURL="https://s3.ca-central-1.amazonaws.com/cli-update/update/state/unstable/"
# Name of the executable to ultimately use.
STATEEXE="state"
# ID of the $PATH entry in the user's ~/.profile for the executable.
STATEID="ActiveStateCLI"

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

# Process command line arguments.
while getopts "nb:?h" opt; do
  case $opt in
  b)
    STATEURL=`echo $STATEURL | sed -e "s/unstable/$OPTARG/;"`
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

# Determine the current OS.
case `uname -s` in
Linux)
  os=linux
  sha256sum="sha256sum"
  ;;
*BSD)
  os=`uname -s | tr '[A-Z]' '[a-z]'`
  sha256sum=""
  error "BSDs not supported yet"
  exit 1
  ;;
Darwin)
  os=darwin
  sha256sum="shasum -a 256"
  ;;
*)
  error "Unsupported OS: `uname -s`"
  exit 1
  ;;
esac

# Determine the current architecture.
case `uname -m` in
i?86)
  arch=386
  ;;
x86_64)
  arch=amd64
  ;;
esac

# Determine the tmp directory.
if [ -z "$TMPDIR" ]; then
  TMPDIR="/tmp"
fi

# Construct system-dependent filenames.
statejson=$os-$arch.json
statepkg=$os-$arch.gz
stateexe=$os-$arch

info "${PREFIX}Preparing for installation...${SUFFIX}"

# Determine a fetch method
if [ ! -z "`type -t wget 2>/dev/null`" ]; then
  fetch="wget -nv -O"
elif [ ! -z "`type -t curl 2>/dev/null`" ]; then
  fetch="curl -vsS -o"
else
  error "Either wget or curl is required to download files"
  exit 1
fi

if [ ! -f $TMPDIR/$statepkg -a ! -f $TMPDIR/$stateexe ]; then
  info "Determining latest version..."
  # Determine the latest version to fetch.
  $fetch $TMPDIR/$statejson $STATEURL$statejson || exit 1
  version=`cat $TMPDIR/$statejson | grep -m 1 '"Version":' | awk '{print $2}' | tr -d '",'`
  rm $TMPDIR/$statejson
  if [ -z "$version" ]; then
    error "Unable to retrieve the latest version number"
    exit 1
  fi
  info "Fetching the latest version: $version..."
  # Fetch it.
  $fetch $TMPDIR/$statepkg ${STATEURL}${version}/${statepkg} || exit 1
fi

# Extract the State binary after verifying its checksum.
if [ -f $TMPDIR/$statepkg ]; then
  # Verify checksum.
  info "Verifying checksum..."
  shasum=`$fetch - $STATEURL$statejson | grep -m 1 '"Sha256":' | awk '{print $2}' | tr -d '",'`
  if [ "`$sha256sum -b $TMPDIR/$statepkg | cut -d ' ' -f1`" != "$shasum" ]; then
    error "SHA256 sum did not match:"
    error "Expected: $shasum"
    error "Received: `$sha256sum -b $TMPDIR/$statepkg | cut -d ' ' -f1`"
    error "Aborting installation."
    exit 1
  fi

  info "Extracting $statepkg..."
  gunzip $TMPDIR/$statepkg || exit 1
  chmod +x $TMPDIR/$stateexe
fi

# Check for existing installation. Otherwise, make the installation default to
# /usr/local/bin if the user has write permission, or to a local bin.
installdir="`dirname \`which $STATEEXE\` 2>/dev/null`"
if [ ! -z "$installdir" ]; then
  warn "Previous installation detected at $installdir"
else
  if [ -w "/usr/local/bin" ]; then
    installdir="/usr/local/bin"
  else
    installdir="$HOME/.local/bin"
  fi
fi

# Prompt the user for a directory to install to.
while "true"; do
  userprompt "Please enter the installation directory [$installdir]: "
  input=$(userinput $installdir)
  if [ -e "$input" -a ! -d "$input" ]; then
    warn "$input exists and is not a directory"
    continue
  elif [ -e "$input" -a ! -w "$input" ]; then
    warn "You do not have permission to write to $input"
    continue
  fi
  if [ ! -z "$input" ]; then
    if [ ! -z "`realpath \"$input\" 2>/dev/null`" ]; then
      installdir="`realpath \"$input\"`"
    else
      installdir="$input"
    fi
  fi
  info "Installing to $installdir"
  if [ ! -e "$installdir" ]; then
    info "NOTE: $installdir will be created"
  elif [ -e "$installdir/$STATEEXE" ]; then
    warn "WARNING: overwriting previous installation"
  fi
  if [ ! -z "`which $STATEEXE`" -a "`dirname \`which $STATEEXE\` 2>/dev/null`" != "$installdir" ]; then
    warn "WARNING: installing elsewhere from previous installation"
  fi
  userprompt "Continue? [y/N/q] "
  response=$(userinput y)
  case "$response" in
    [Qq])
      error "Aborting installation"
      exit 0
      ;;
    [Yy])
      # Install.
      if [ ! -e "$installdir" ]; then
        mkdir -p "$installdir" || continue
      fi
      info "Installing to $installdir..."
      mv $TMPDIR/$stateexe "$installdir/$STATEEXE"
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
if [ "`dirname \`which $STATEEXE\` 2>/dev/null`" = "$installdir" ]; then
  info "Installation complete."
  info "You may now start using the '$STATEEXE' program."
  exit 0
fi
profile="`info $HOME`/.profile"
if [ ! -w "$profile" ]; then
  info "Installation complete."
  echo "Please manually add $installdir to your \$PATH in order to start "
  echo "using the '$STATEEXE' program."
  exit 0
fi
userprompt "Allow \$PATH to be appended to in your $profile? [y/N]"
response=$(userinput y | tr '[:upper:]' '[:lower:]')
if [ "$response" != "y" ]; then
  info "Installation complete."
  echo "Please manually add $installdir to your \$PATH in order to start "
  echo "using the '$STATEEXE' program."
  exit 0
fi
info "Updating environment..."
pathenv="export PATH=\"\$PATH:$installdir\" #$STATEID"
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
