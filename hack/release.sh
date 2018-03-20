#!/usr/bin/env bash

# This script will pull down an existing build and tag it for release.
# Example: SHA=492978... RELEASE=0.1.0 hack/release.sh

set -e

RELEASE=${RELEASE:=0.1.0}

SHA=${SHA:=$(git rev-parse HEAD)}

OPERATOR_IMAGE=wcr.io/oracle/mysql-operator
AGENT_IMAGE=wcr.io/oracle/mysql-agent

function do_release() {
    echo "Creating release $RELEASE from existing version $SHA"

    if git rev-parse "$RELEASE" >/dev/null 2>&1; then
        echo "Tag $RELEASE already exists. Doing nothing."
        exit 1
    fi

    echo "Creating images"
    docker pull $OPERATOR_IMAGE:$SHA
    docker tag $OPERATOR_IMAGE:$SHA $OPERATOR_IMAGE:$RELEASE
    docker push $OPERATOR_IMAGE:$RELEASE

    docker pull $AGENT_IMAGE:$SHA
    docker tag $AGENT_IMAGE:$SHA $AGENT_IMAGE:$RELEASE
    docker push $AGENT_IMAGE:$RELEASE

    echo "Creating release tag $RELEASE"
    git tag -a "$RELEASE" -m "Release version: $RELEASE"
    git push --tags
}

read -r -p "Are you sure you want to release ${SHA} as ${RELEASE}? [y/N] " response
case "$response" in
    [yY][eE][sS]|[yY])
        do_release
        ;;
    *)
        exit
        ;;
esac
