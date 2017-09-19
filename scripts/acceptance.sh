#!/bin/bash -ex

PY_REQUIRES="requests==2.18.4 robotframework==3.0.2"

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/../

if [ "$(uname)" == "Darwin" ]; then
    PLATFORM="darwin"
else
    PLATFORM="linux"
fi

export PATH="$PWD/testbin:$PWD/bin/$PLATFORM/amd64:$PATH"

export HELM_HOME="$PWD/.helm"
helm init --client-only

if [ ! -d .venv/ ]; then
    virtualenv -p $(which python2.7) .venv/
    .venv/bin/python .venv/bin/pip install $PY_REQUIRES
fi

mkdir -p .robot/
.venv/bin/robot --outputdir=.robot/ acceptance_tests/
