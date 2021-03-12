#!/bin/bash -ex

HELM_VERSION="2.17.0"

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/../

export PATH="$PWD/testbin:$PATH"

main() {
    export HELM_HOME="$PWD/.helm"
    install_helm
    package_test_charts
}

install_helm() {
    if [ ! -f "testbin/helm" ]; then
        mkdir -p testbin/
        [ "$(uname)" == "Darwin" ] && PLATFORM="darwin" || PLATFORM="linux"
        TARBALL="helm-v${HELM_VERSION}-${PLATFORM}-amd64.tar.gz"
        wget "https://storage.googleapis.com/kubernetes-helm/${TARBALL}" || \
          curl -O "https://storage.googleapis.com/kubernetes-helm/${TARBALL}"
        tar -C testbin/ -xzf $TARBALL
        rm -f $TARBALL
        pushd testbin/
        UNCOMPRESSED_DIR="$(find . -mindepth 1 -maxdepth 1 -type d)"
        mv $UNCOMPRESSED_DIR/helm .
        rm -rf $UNCOMPRESSED_DIR
        chmod +x ./helm
        popd
        helm init --client-only

        # remove any repos that come out-of-the-box (i.e. "stable")
        helm repo list | sed -n '1!p' | awk '{print $1}' | xargs helm repo remove
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
    popd

    pushd testdata/badcharts/
    for d in $(find . -maxdepth 1 -mindepth 1 -type d); do
        pushd $d
        # TODO: remove in v0.14.0. We do not generate .prov file for this chart
        # since prov validation is not enabled, and it breaks acceptance tests
        if grep "mybadsemver2chart" Chart.yaml; then
            helm package . || true
        fi
        popd
    done
}

main
