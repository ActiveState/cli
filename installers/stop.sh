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
ps ax | grep -i "state-installer\|activestate.*state" | awk '{ print $1;}' | xargs kill -9

echo "Done"