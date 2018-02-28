#!/bin/sh
# Copyright 2018 ActiveState Software Inc. All rights reserved.

# URL to fetch updates from.
STATEURL="https://s3.ca-central-1.amazonaws.com/cli-update/update/state/"
# Name of the executable to ultimately
STATEEXE="state"
# ID of the $PATH entry in the user's ~/.profile for the executable.
STATEID="ActiveState"

# Determine the name of the State binary package to install.
case `uname -m` in
i?86)
  statejson=linux-386.json
  statepkg=linux-386.gz
  stateexe=linux-386
  ;;
x86_64)
  statejson=linux-amd64.json
  statepkg=linux-amd64.gz
  stateexe=linux-amd64
  ;;
*)
  echo "Unknown architecture: `uname -m`"
  exit 1
;;
esac

echo "Preparing for installation..."

if [ ! -f $statepkg ]; then
  # Determine the latest version to fetch.
  version=`wget -q -O - $STATEURL$statejson | grep -m 1 '"Version":' | awk '{print $2}' | tr -d '",'`
  echo "Fetching the latest version: $version"
  # Fetch it.
  wget -q ${STATEURL}${version}/${statepkg} || exit 1
fi

# Extract the State binary.
echo "Extracting $statepkg..."
gunzip $statepkg

# Verify checksum.
echo "Verifying checksum..."
shasum=`wget -q -O - $STATEURL$statejson | grep -m 1 '"Sha256":' | awk '{print $2}' | tr -d '",'`
if [ "`sha256sum -b $stateexe | cut -d ' ' -f1`" != "$shasum" ]; then
  echo "SHA256 sum did not match:"
  echo "Expected: $shasum"
  echo "Received: `sha256sum -b $stateexe | cut -d ' ' -f1`"
  echo "Aborting installation."
  exit 1
fi

# Prompt the user for a directory to install to.
installdir="`pwd`"
while "true"; do
  echo -n "Please enter the installation directory [$installdir]: "
  read input
  if [ -e "$input" -a ! -d "$input" ]; then
    echo "$input exists and is not a directory."
    continue
  fi
  if [ ! -z "$input" ]; then
    installdir="`realpath \"$input\"`"
  fi
  echo "Installing to $installdir"
  echo -n "Continue? [Y/n/q] "
  read response
  case "$response" in
    [Qq])
      echo "Aborting installation"
      exit 0
      ;;
    [Yy])
      # Install.
      echo "Installing to $installdir..."
      mv $stateexe "$installdir/$STATEEXE"
      if [ $? -eq 0 ]; then
        break
      fi
      ;;
    [Nn]|*)
      continue
      ;;
  esac
done

# Update user's profile and the current environment.
echo "Updating environment..."
profile="`echo $HOME`/.profile"
pathenv="export PATH=\"\$PATH:$installdir\" #ActiveState"
if [ -z "`grep -no \"\#$STATEID\" \"$profile\"`" ]; then
  echo "Adding to \$PATH in $profile"
  echo "\n$pathenv" >> "$profile"
else
  echo "Updating \$PATH in $profile"
  sed -i -e "s|^export PATH=[^\#]\+\#$STATEID|$pathenv|;" "$profile"
fi

echo "Done."
echo "Please either run 'source ~/.profile' or start a new login shell in "
echo "order to complete the installation."
