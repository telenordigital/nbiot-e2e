#!/bin/bash

usage() {
  echo -n "usage: ${0} proj_dir

Check proj_dir for git origin remote changes. When directory is behind origin,
pull and exit 0. Otherwise exit 1."
}

if [[ "$#" -lt 1 ]]
then
  echo Too few arguments
  usage
  exit 1
fi

DIR=$1
cd "$DIR"
echo "checking $(git remote get-url origin) for changes"

git remote update
git status | grep behind || exit

set -e

echo "origin has changed"
echo "git pull"
git pull
exit $?
