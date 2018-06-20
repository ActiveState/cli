#!/usr/bin/env bash

echo "STABLE_BRANCHNAME "  `git rev-parse --abbrev-ref HEAD`
echo "STABLE_BUILDNUMBER " `git rev-list --abbrev-commit HEAD | wc -l`
echo "STABLE_REVISIONHASH" `git rev-parse --verify HEAD`

