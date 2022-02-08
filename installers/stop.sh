#!/usr/bin/env sh
# Copyright 2022 ActiveState Software Inc. All rights reserved.

if [ "$EUID" -ne 0 ]; then
  echo "Please run this script as administrator. This is required to ensure any existing State Tool processes are terminated."
  exit 1
fi

read -r -p "You are about to kill all State Tool processes, continue? (y/n)" response
if ! [ "$response" != "${response#[Yy]}" ] ;then
    echo "Cancelled"
    exit 1
fi

echo "Stopping running State Tool processes"
# Because POSIX doesn't always give the executable path of a process we have to get a little creative with lsof here.
# Which isn't as reliable as checking the executable path because it gives all files in use by the process.
# Therefore we also check that the en-us.yaml is in use, which helps build confidence that we are killing the right process.
for PID in `ps ax | grep -i "state-installer\|state" | grep -v "grep" | awk '{ print $1;}'`; do
  LSOUT=`lsof -p $PID`
  if [ "`echo $LSOUT | grep -i "activestate"`" != "" ]; then
    if [ "`basename $(ps -p $PID -o comm=)`" == "state" ] && [ "`echo $LSOUT | grep -i "en-us.yaml"`" == "" ]; then
      continue
    fi
    kill -9 $PID
  fi
done

echo "Done"