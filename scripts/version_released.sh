#!/bin/bash -ex

VERSION="$1"

REQUIRED_RELEASE_ENV_VARS=(
    "RELEASE_AMAZON_BUCKET"
    "RELEASE_AMAZON_REGION"
)

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/../

main() {
    check_args
    check_env_vars
    version_released
}

check_args() {
    if [ "$VERSION" == "" ]; then
        echo "usage: is_released.sh <version>"
    fi
}

check_env_vars() {
    set +x
    ALL_ENV_VARS_PRESENT="1"
    for VAR in ${REQUIRED_RELEASE_ENV_VARS[@]}; do
           if [ "${!VAR}" == "" ]; then
            echo "missing required test env var: $VAR"
            ALL_ENV_VARS_PRESENT="0"
        fi
    done
    if [ "$ALL_ENV_VARS_PRESENT" == "0" ]; then
        exit 1
    fi
    set -x
}

version_released() {
    aws s3 --region=$RELEASE_AMAZON_REGION ls s3://$RELEASE_AMAZON_BUCKET/release/ \
        | grep -F "v${VERSION}/" >/dev/null
}

main
