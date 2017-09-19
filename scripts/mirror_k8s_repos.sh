#!/bin/bash -ex

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/../

trap "rm -f index.yaml" EXIT
mkdir -p mirror/

get_all_tgzs() {
    local repo_url="$1"
    rm -f index.yaml
    wget $repo_url/index.yaml
    tgzs="$(ruby -ryaml -e \
        "YAML.load_file('index.yaml')['entries'].each do |k,e|;for c in e;puts c['urls'][0];end;end")"
    pushd mirror/
    for tgz in $tgzs; do
        if [ ! -f "${tgz##*/}" ]; then
            wget $tgz
        fi
    done
    popd
}

# Stable
get_all_tgzs https://kubernetes-charts.storage.googleapis.com

# Incubator
get_all_tgzs https://kubernetes-charts-incubator.storage.googleapis.com
