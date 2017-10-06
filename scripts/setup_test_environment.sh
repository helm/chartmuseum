#!/bin/bash -ex

HELM_VERSION="2.6.2"
REQUIRED_TEST_ENV_VARS=(
    "TEST_STORAGE_AMAZON_BUCKET"
    "TEST_STORAGE_AMAZON_REGION"
    "TEST_STORAGE_GOOGLE_BUCKET"
)

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/../

export PATH="$PWD/testbin:$PATH"
export HELM_HOME="$PWD/.helm"

main() {
    check_env_vars
    install_helm
    package_test_charts
}

check_env_vars() {
    set +x
    ALL_ENV_VARS_PRESENT="1"
    for VAR in ${REQUIRED_TEST_ENV_VARS[@]}; do
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

install_helm() {
    if [ ! -f "testbin/helm" ]; then
        mkdir -p testbin/
        [ "$(uname)" == "Darwin" ] && PLATFORM="darwin" || PLATFORM="linux"
        TARBALL="helm-v${HELM_VERSION}-${PLATFORM}-amd64.tar.gz"
        wget "https://storage.googleapis.com/kubernetes-helm/${TARBALL}"
        tar -C testbin/ -xzf $TARBALL
        rm -f $TARBALL
        pushd testbin/
        UNCOMPRESSED_DIR="$(find . -mindepth 1 -maxdepth 1 -type d)"
        mv $UNCOMPRESSED_DIR/helm .
        rm -rf $UNCOMPRESSED_DIR
        chmod +x ./helm
        popd
        helm init --client-only
    fi
}

package_test_charts() {
    pushd testdata/charts/
    for d in $(find . -maxdepth 1 -mindepth 1 -type d); do
        pushd $d
        helm package --sign --key helm-test --keyring ../../pgp/helm-test-key.secret .
        popd
    done
    popd
}

main
