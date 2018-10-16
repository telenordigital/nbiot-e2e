#!/bin/bash

usage() {
  echo -n "usage: ${0} proj_dir command

Check proj_dir for git origin remote changes. When directory is behind origin,
pull and execute command."
}

if [[ "$#" -lt 2 ]]
then
  echo Too few arguments
  usage
  exit 1
fi

# TODO implement help text
DIR=$1
shift
CMD=$@

cd "$DIR"

git remote update
git status | grep behind

if [ $? -gt 0 ]
then
  exit 0
fi

set -e

echo "origin has changed - updating"
echo git pull --rebase

$CMD
