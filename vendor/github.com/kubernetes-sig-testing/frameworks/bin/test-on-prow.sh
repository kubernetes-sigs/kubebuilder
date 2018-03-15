#!/usr/bin/env bash

set -e
set -u

./bin/install-test-dependencies.sh
./bin/pre-commit.sh
