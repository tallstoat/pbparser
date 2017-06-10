#!/bin/bash

# Fail the script on error
set -e

read -p "Semantic version (without the 'v'): " VERSION

if [[ $VERSION == v* ]]
then
  echo ERROR: Please specify the raw semantic version without a 'v' prefix eg. x.y.z
  exit 1
fi


REVISION=`git rev-parse --short HEAD`
echo Starting to tag $REVISION as v$VERSION...

git tag -a v$VERSION -m "Release v$VERSION"
if [ $? -eq 1 ]; then
  echo "Failed to tag; exiting..."
  exit 1
fi

echo Finished!
echo "NOTE: Remember to execute (when satisfied): git push origin v$VERSION"
echo
