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

set -e

git remote update
#git status | grep behind || exit
status=$(git status)
if [[ "$status" =~ "have diverged" ]]; then
  echo "origin has diverged"
  echo "git fetch"
  git fetch
  # get remote branch ref
  ref=$(git rev-parse --abbrev-ref --symbolic-full-name @{u})
  echo "git reset --hard $ref"
  git reset --hard $ref
elif [[ "$status" =~ "behind" ]]; then
  echo "origin has changed"
  echo "git pull"
  git pull
else
  exit 1
fi

exit $?
