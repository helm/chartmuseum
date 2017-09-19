#!/bin/bash -ex

VERSION="$1"
DOCKER_REPO="chartmuseum/chartmuseum"
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
    docker_build
    release_latest
    release_stable
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

docker_build() {
    docker build -t $DOCKER_REPO:latest .
}

release_latest() {
    echo "$COMMIT" > .latest.txt
    aws s3 --region=$RELEASE_AMAZON_REGION cp --recursive bin/ \
        s3://$RELEASE_AMAZON_BUCKET/release/latest/bin/
    aws s3 --region=$RELEASE_AMAZON_REGION cp .latest.txt \
        s3://$RELEASE_AMAZON_BUCKET/release/latest.txt
    docker push $DOCKER_REPO:latest
}

release_stable() {
    set +e
    aws s3 --region=$RELEASE_AMAZON_REGION ls s3://$RELEASE_AMAZON_BUCKET/release/ \
        | grep -F "v${VERSION}/"
    local rc="$?"
    set -e
    if [ "$rc" == "0" ]; then
        echo "v${VERSION} has already been released. Skipping."
    else
        echo "v${VERSION}" > .stable.txt
        aws s3 --region=$RELEASE_AMAZON_REGION cp --recursive bin/ \
            s3://$RELEASE_AMAZON_BUCKET/release/v${VERSION}/bin/
        aws s3 --region=$RELEASE_AMAZON_REGION cp .stable.txt \
            s3://$RELEASE_AMAZON_BUCKET/release/stable.txt
        docker tag $DOCKER_REPO:latest $DOCKER_REPO:v${VERSION}
        docker push $DOCKER_REPO:v${VERSION}
    fi
}

main
