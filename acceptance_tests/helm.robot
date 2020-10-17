*** Settings ***
Documentation     Tests to verify that ChartMuseum is able to work with
...               Helm CLI and act as a valid Helm Chart Repository using
...               all supported storage backends (local, s3, gcs, openstack).
Library           String
Library           OperatingSystem
Library           lib/ChartMuseum.py
Library           lib/Helm.py
Suite Setup       Suite Setup
Suite Teardown    Suite Teardown

*** Test Cases ***
ChartMuseum works with Helm using local storage
    Test Helm integration   local

ChartMuseum works with Helm using Amazon cloud storage
    Test Helm integration   amazon

ChartMuseum works with Helm using Google cloud storage
    Test Helm integration   google

ChartMuseum works with Helm using Microsoft cloud storage
    Test Helm integration   microsoft

ChartMuseum works with Helm using Alibaba cloud storage
    Test Helm integration   alibaba

ChartMuseum works with Helm using Openstack cloud storage
    Test Helm integration   openstack

ChartMuseum works with Helm using Oracle cloud storage
    Test Helm integration   oracle

ChartMuseum works with Helm using Baidu cloud storage
    Test Helm integration   baidu

*** Keyword ***
Test Helm integration
    [Arguments]    ${storage}

    # return fast if we cannot find a bucket/container in an environment variable.
    ${USTORAGE}=  Convert To Uppercase  ${storage}
    ${ENV_STORAGE_BUCKET_SET}=  Get Environment variable  TEST_STORAGE_${USTORAGE}_BUCKET  ${EMPTY}
    Return from Keyword if  '${ENV_STORAGE_BUCKET_SET}'=='${EMPTY}' and '${storage}'!='local' and '${storage}'!='microsoft' and '${storage}'!='openstack'
    ${ENV_STORAGE_CONTAINER_SET}=  Get Environment variable  TEST_STORAGE_${USTORAGE}_CONTAINER  ${EMPTY}
    Return from Keyword if  '${ENV_STORAGE_CONTAINER_SET}'=='${EMPTY}' and ('${storage}'=='microsoft' or '${storage}'=='openstack')
    ${ENV_STORAGE_CONTAINER_SET}=  Get Environment variable  TEST_STORAGE_${USTORAGE}_CONTAINER  ${EMPTY}

    Start ChartMuseum server with storage backend  ${storage}
    Able to add ChartMuseum as Helm chart repo
    Helm search does not return test charts
    Unable to fetch and verify test charts
    Upload test charts to ChartMuseum
    Upload provenance files to ChartMuseum
    Upload bad test charts to ChartMuseum
    Upload bad provenance files to ChartMuseum
    Able to update ChartMuseum repo
    Helm search returns test charts
    Able to fetch and verify test charts
    Delete test charts from ChartMuseum
    Able to update ChartMuseum repo
    Ensure test charts deleted
    Helm search does not return test charts
    Unable to fetch and verify test charts

Start ChartMuseum server with storage backend
    [Arguments]    ${storage}
    ChartMuseum.start chartmuseum  ${storage}
    ChartMuseum.wait for chartmuseum

Upload test charts to ChartMuseum
    ChartMuseum.upload test charts

Upload provenance files to ChartMuseum
    ChartMuseum.upload provenance files

Upload bad test charts to ChartMuseum
    ChartMuseum.upload bad test charts

Upload bad provenance files to ChartMuseum
    ChartMuseum.upload bad provenance files

Delete test charts from ChartMuseum
    ChartMuseum.delete test charts

Ensure test charts deleted
    ChartMuseum.ensure charts deleted

Able to add ChartMuseum as Helm chart repo
    Helm.add chart repo
    Helm.return code should be  0
    Helm.output contains  has been added

Able to update ChartMuseum repo
    Helm.update chart repos
    Helm.return code should be  0

Helm search returns test charts
    Helm.search for chart  mychart
    Helm.output contains  mychart

Helm search does not return test charts
    Helm.search for chart  mychart
    Helm.output does not contain  mychart
    Helm.search for chart  mychart2
    Helm.output does not contain  mychart2

Able to fetch and verify test charts
    Helm.fetch and verify chart  mychart
    Helm.return code should be  0

Unable to fetch and verify test charts
    Helm.fetch and verify chart  mychart
    Helm.return code should not be  0

Suite Setup
    ChartMuseum.remove chartmuseum logs

Suite Teardown
    Helm.remove chart repo
    ChartMuseum.stop chartmuseum
    ChartMuseum.print chartmuseum logs
