#!/bin/bash -ex

VERSION="$1"

REQUIRED_RELEASE_ENV_VARS=(
    "RELEASE_AMAZON_BUCKET"
    "RELEASE_AMAZON_REGION"
)

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/../

COMMIT="$(git rev-parse HEAD)"

main() {
    check_args
    check_env_vars

    if [ "$VERSION" == "latest" ]; then
        release_latest
    else
        release_stable
    fi
}

check_args() {
    if [ "$VERSION" == "" ]; then
        echo "usage: release.sh <version>"
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

release_latest() {
    echo "$COMMIT" > .latest.txt
    aws s3 --region=$RELEASE_AMAZON_REGION cp --recursive bin/ \
        s3://$RELEASE_AMAZON_BUCKET/release/latest/bin/
    aws s3 --region=$RELEASE_AMAZON_REGION cp .latest.txt \
        s3://$RELEASE_AMAZON_BUCKET/release/latest.txt
}

release_stable() {
    echo "v${VERSION}" > .stable.txt
    aws s3 --region=$RELEASE_AMAZON_REGION cp --recursive bin/ \
        s3://$RELEASE_AMAZON_BUCKET/release/v${VERSION}/bin/
    aws s3 --region=$RELEASE_AMAZON_REGION cp .stable.txt \
        s3://$RELEASE_AMAZON_BUCKET/release/stable.txt
}

main
