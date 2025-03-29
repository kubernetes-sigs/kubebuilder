#!/bin/bash

# Color codes
readonly reset=$(tput sgr0)
readonly green=$(tput bold; tput setaf 2)
readonly yellow=$(tput bold; tput setaf 3)
readonly blue=$(tput bold; tput setaf 6)
readonly timeout=$(if [ "$(uname)" == "Darwin" ]; then echo "1"; else echo "0.1"; fi)

# Function to display descriptions
function desc() {
 maybe_first_prompt
 rate=25
 if [ -n "$DEMO_RUN_FAST" ]; then
  rate=1000
 fi
 echo "$blue# $@$reset" | pv -qL $rate
 prompt
}

# Function to display prompt
function prompt() {
 echo -n "$yellow\$ $reset"
}

# Function to clear if not asciinema (Add this function)
function clear_if_not_asciinema() {
    if ! $is_asciinema_running; then
        clear
    fi
}

started=""
function maybe_first_prompt() {
 if [ -z "$started" ]; then
  prompt
  started=true
 fi
}

# After a `run` this variable will hold the stdout of the command that was run
DEMO_RUN_STDOUT=""


