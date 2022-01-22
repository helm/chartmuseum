import contextlib
import glob
import os
import requests
import shutil
import socket
import time

import common


class ChartMuseum(common.CommandRunner):
    def http_status_code_should_be(self, expected_status, actual_status):
        if int(expected_status) != int(actual_status):
            raise AssertionError('Expected HTTP status code to be %s but was %s.'
                                 % (expected_status, actual_status))

    def start_chartmuseum(self, storage):
        self.stop_chartmuseum()
        os.chdir(self.rootdir)
        cmd = 'KILLME=1 chartmuseum --debug --port=%d --storage="%s" ' % (common.PORT, storage)
        if storage == 'local':
            shutil.rmtree(common.STORAGE_DIR, ignore_errors=True)
            cmd += '--storage-local-rootdir=%s >> %s 2>&1' % (common.STORAGE_DIR, common.LOGFILE)
        elif storage == 'amazon':
            cmd += '--storage-amazon-bucket="%s" --storage-amazon-prefix="%s" --storage-amazon-region="%s" >> %s 2>&1' \
                  % (common.STORAGE_AMAZON_BUCKET, common.STORAGE_AMAZON_PREFIX, common.STORAGE_AMAZON_REGION, common.LOGFILE)
        elif storage == 'google':
            cmd += '--storage-google-bucket="%s" --storage-google-prefix="%s" >> %s 2>&1' \
                   % (common.STORAGE_GOOGLE_BUCKET, common.STORAGE_GOOGLE_PREFIX, common.LOGFILE)
        elif storage == 'microsoft':
            cmd += '--storage-microsoft-container="%s" --storage-microsoft-prefix="%s"  >> %s 2>&1' \
                   % (common.STORAGE_MICROSOFT_CONTAINER, common.STORAGE_MICROSOFT_PREFIX, common.LOGFILE)
        elif storage == 'alibaba':
            cmd += '--storage-alibaba-bucket="%s" --storage-alibaba-prefix="%s" --storage-alibaba-endpoint="%s" >> %s 2>&1' \
                  % (common.STORAGE_ALIBABA_BUCKET, common.STORAGE_ALIBABA_PREFIX, common.STORAGE_ALIBABA_ENDPOINT, common.LOGFILE)
        elif storage == 'openstack':
            cmd += '--storage-openstack-container="%s" --storage-openstack-prefix="%s" --storage-openstack-region="%s" >> %s 2>&1' \
                  % (common.STORAGE_OPENSTACK_CONTAINER, common.STORAGE_OPENSTACK_PREFIX, common.STORAGE_OPENSTACK_REGION, common.LOGFILE)
        elif storage == 'oracle':
            cmd += '--storage-oracle-bucket="%s" --storage-oracle-prefix="%s" --storage-oracle-compartmentid="%s" >> %s 2>&1' \
                   % (common.STORAGE_ORACLE_BUCKET, common.STORAGE_ORACLE_PREFIX, common.STORAGE_ORACLE_COMPARTMENTID, common.LOGFILE)
        elif storage == 'baidu':
            cmd += '--storage-baidu-bucket="%s" --storage-baidu-prefix="%s" --storage-baidu-endpoint="%s" >> %s 2>&1' \
                   % (common.STORAGE_BAIDU_BUCKET, common.STORAGE_BAIDU_PREFIX, common.STORAGE_BAIDU_ENDPOINT, common.LOGFILE)
        print(cmd)
        self.run_command(cmd, detach=True)

    def wait_for_chartmuseum(self):
        seconds_waited = 0
        while True:
            with contextlib.closing(socket.socket(socket.AF_INET, socket.SOCK_STREAM)) as sock:
                result = sock.connect_ex(('localhost', common.PORT))
            if result == 0:
                break
            if seconds_waited == common.MAX_WAIT_SECONDS:
                raise Exception('Reached max time (%d seconds) waiting for chartmuseum to come up' % common.MAX_WAIT_SECONDS)
            time.sleep(1)
            seconds_waited += 1

    def stop_chartmuseum(self):
        self.run_command('pkill -9 chartmuseum')
        shutil.rmtree(common.STORAGE_DIR, ignore_errors=True)

    def remove_chartmuseum_logs(self):
        os.chdir(self.rootdir)
        self.run_command('rm -f %s' % common.LOGFILE)

    def print_chartmuseum_logs(self):
        os.chdir(self.rootdir)
        self.run_command('cat %s' % common.LOGFILE)

    def upload_test_charts(self):
        charts_endpoint = '%s/api/charts' % common.HELM_REPO_URL
        testcharts_dir = os.path.join(self.rootdir, common.TESTCHARTS_DIR)
        os.chdir(testcharts_dir)
        for d in os.listdir('.'):
            if not os.path.isdir(d):
                continue
            os.chdir(d)
            tgzs = glob.glob('*.tgz')
            for tgz in tgzs:
                print(('Uploading test chart package "%s"' % tgz))
                with open(tgz, 'rb') as f:
                    response = requests.post(url=charts_endpoint, data=f.read())
                    print(('POST %s' % charts_endpoint))
                    print(('HTTP STATUS: %s' % response.status_code))
                    print(('HTTP CONTENT: %s' % response.content))
                    self.http_status_code_should_be(201, response.status_code)
            os.chdir('../')

    def upload_bad_test_charts(self):
        charts_endpoint = '%s/api/charts' % common.HELM_REPO_URL
        testcharts_dir = os.path.join(self.rootdir, common.TESTBADCHARTS_DIR)
        os.chdir(testcharts_dir)
        for d in os.listdir('.'):
            if not os.path.isdir(d):
                continue
            os.chdir(d)
            tgzs = glob.glob('*.tgz')
            for tgz in tgzs:
                print(('Uploading bad test chart package "%s"' % tgz))
                with open(tgz, 'rb') as f:
                    response = requests.post(url=charts_endpoint, data=f.read())
                    print(('POST %s' % charts_endpoint))
                    print(('HTTP STATUS: %s' % response.status_code))
                    print(('HTTP CONTENT: %s' % response.content))
                    
                    self.http_status_code_should_be(400, response.status_code)
            os.chdir('../')

    def upload_provenance_files(self):
        prov_endpoint = '%s/api/prov' % common.HELM_REPO_URL
        testcharts_dir = os.path.join(self.rootdir, common.TESTCHARTS_DIR)
        os.chdir(testcharts_dir)
        for d in os.listdir('.'):
            if not os.path.isdir(d):
                continue
            os.chdir(d)
            provs = glob.glob('*.tgz.prov')
            for prov in provs:
                print(('Uploading provenance file "%s"' % prov))
                with open(prov) as f:
                    response = requests.post(url=prov_endpoint, data=f.read())
                    print(('POST %s' % prov_endpoint))
                    print(('HTTP STATUS: %s' % response.status_code))
                    print(('HTTP CONTENT: %s' % response.content))
                    self.http_status_code_should_be(201, response.status_code)
            os.chdir('../')

    def upload_bad_provenance_files(self):
        prov_endpoint = '%s/api/prov' % common.HELM_REPO_URL
        testcharts_dir = os.path.join(self.rootdir, common.TESTBADCHARTS_DIR)
        os.chdir(testcharts_dir)
        for d in os.listdir('.'):
            if not os.path.isdir(d):
                continue
            os.chdir(d)
            provs = glob.glob('*.tgz.prov')
            for prov in provs:
                print(('Uploading bad provenance file "%s"' % prov))
                with open(prov) as f:
                    response = requests.post(url=prov_endpoint, data=f.read())
                    print(('POST %s' % prov_endpoint))
                    print(('HTTP STATUS: %s' % response.status_code))
                    print(('HTTP CONTENT: %s' % response.content))
                    self.http_status_code_should_be(400, response.status_code)
            os.chdir('../')

    def delete_test_charts(self):
        endpoint = '%s/api/charts' % common.HELM_REPO_URL
        testcharts_dir = os.path.join(self.rootdir, common.TESTCHARTS_DIR)
        os.chdir(testcharts_dir)
        for d in os.listdir('.'):
            # delete all charts inside /mychart (also includes mychart2)
            if not os.path.isdir(d):
                continue
            os.chdir(d)
            tgzs = glob.glob('*.tgz')
            for tgz in tgzs:
                tmp = tgz[:-4].rsplit('-', 1)
                name = tmp[0]
                version = tmp[1]
                print(('Delete test chart "%s-%s"' % (name, version)))
                with open(tgz) as f:
                    epoint = '%s/%s/%s' % (endpoint, name, version)
                    response = requests.delete(url=epoint)
                    print(('HTTP STATUS: %s' % response.status_code))
                    print(('HTTP CONTENT: %s' % response.content))
                    self.http_status_code_should_be(200, response.status_code)
            os.chdir('../')

    def ensure_charts_deleted(self):
        endpoint = '%s/api/charts' % common.HELM_REPO_URL
        testcharts_dir = os.path.join(self.rootdir, common.TESTCHARTS_DIR)
        os.chdir(testcharts_dir)
        for d in os.listdir('.'):
            if not os.path.isdir(d):
                continue
            os.chdir(d)
            tgzs = glob.glob('*.tgz')
            for tgz in tgzs:
                tmp = tgz[:-4].rsplit('-', 1)
                name = tmp[0]
                version = tmp[1]
                with open(tgz):
                    epoint = '%s/%s/%s' % (endpoint, name, version)
                    response = requests.get(url=epoint)
                    self.http_status_code_should_be(404, response.status_code)
            os.chdir('../')
