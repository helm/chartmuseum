#!/bin/bash -ex

PY_REQUIRES="requests==2.27.1 robotframework==4.1.3"

REQUIRED_TEST_STORAGE_ENV_VARS=(
    "TEST_STORAGE_AMAZON_BUCKET"
    "TEST_STORAGE_AMAZON_REGION"
    "TEST_STORAGE_GOOGLE_BUCKET"
    "TEST_STORAGE_MICROSOFT_CONTAINER"
    "TEST_STORAGE_ALIBABA_BUCKET"
    "TEST_STORAGE_ALIBABA_ENDPOINT"
    "TEST_STORAGE_OPENSTACK_CONTAINER"
    "TEST_STORAGE_OPENSTACK_REGION"
    "TEST_STORAGE_ORACLE_BUCKET"
    "TEST_STORAGE_ORACLE_REGION"
    "TEST_STORAGE_ORACLE_COMPARTMENTID"
)

set +x
for VAR in ${REQUIRED_TEST_STORAGE_ENV_VARS[@]}; do
    if [ "${!VAR}" != "" ]; then
        echo "Detected one required test env var: $VAR"
    fi
done
set -x

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/../

ARCH="amd64"
if [ "$(uname)" == "Darwin" ]; then
    PLATFORM="darwin"
    if [ `uname -m` == 'arm64' ]; then
        ARCH="arm64"
    fi
else
    PLATFORM="linux"
fi

export PATH="$PWD/testbin:$PWD/bin/$PLATFORM/$ARCH:$PWD/_dist/$PLATFORM-$ARCH:$PATH"

mkdir -p .robot/

export XDG_CACHE_HOME=${PWD}/.helm/cache && mkdir -p ${XDG_CACHE_HOME}
export XDG_CONFIG_HOME=${PWD}/.helm/config && mkdir -p ${XDG_CONFIG_HOME}
export XDG_DATA_HOME=${PWD}/.helm/data && mkdir -p ${XDG_DATA_HOME}
if [ ! -d .venv/ ]; then
  virtualenv -p $(which python3) .venv/
  .venv/bin/python .venv/bin/pip install $PY_REQUIRES
fi
.venv/bin/robot --outputdir=.robot/ acceptance_tests/
