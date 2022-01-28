#!/bin/bash -ex

HELM_VERSION="3.8.0"

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/../

export PATH="$PWD/testbin:$PATH"

main() {
    export XDG_CACHE_HOME=${PWD}/.helm/cache && mkdir -p ${XDG_CACHE_HOME}
    export XDG_CONFIG_HOME=${PWD}/.helm/config && mkdir -p ${XDG_CONFIG_HOME}
    export XDG_DATA_HOME=${PWD}/.helm/data && mkdir -p ${XDG_DATA_HOME}
    install_helm
    package_test_charts
}

install_helm() {
    if [ ! -f "testbin/helm" ]; then
        mkdir -p testbin/
        [ "$(uname)" == "Darwin" ] && PLATFORM="darwin" || PLATFORM="linux"
        ARCH="amd64"
        if [ `uname -m` == "arm64" ]; then
            ARCH="arm64"
        fi
        TARBALL="helm-v${HELM_VERSION}-${PLATFORM}-${ARCH}.tar.gz"
        wget "https://get.helm.sh/${TARBALL}" || \
          curl -O "https://get.helm.sh/${TARBALL}"
        tar -C testbin/ -xzf $TARBALL
        rm -f $TARBALL
        pushd testbin/
        UNCOMPRESSED_DIR="$(find . -mindepth 1 -maxdepth 1 -type d)"
        mv $UNCOMPRESSED_DIR/helm .
        rm -rf $UNCOMPRESSED_DIR
        chmod +x ./helm
        popd

        # remove any repos that come out-of-the-box (i.e. "stable")
        helm repo list | sed -n '1!p' | awk '{print $1}' | xargs helm repo remove || true
    fi
}

package_test_charts() {
    pushd testdata/charts/
    for d in $(find . -maxdepth 1 -mindepth 1 -type d); do
        pushd $d
        helm package --sign --key helm-test --keyring ../../pgp/helm-test-key.secret .
        popd
    done
    # add another version to repo for metric tests
    helm package --sign --key helm-test --keyring ../pgp/helm-test-key.secret --version 0.2.0 -d mychart/ mychart/.
    # add another version for per chart limit test
    helm package --sign --key helm-test --keyring ../pgp/helm-test-key.secret --version 0.0.1 -d mychart/ mychart/.
    popd

    pushd testdata/badcharts/
    for d in $(find . -maxdepth 1 -mindepth 1 -type d); do
        pushd $d
        helm package .
        popd
    done
    popd
}

main
