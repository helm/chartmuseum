import glob
import os
import requests
import shutil

import common


class ChartMuseum(common.CommandRunner):
    def http_status_code_should_be(self, expected_status, actual_status):
        if int(expected_status) != int(actual_status):
            raise AssertionError('Expected HTTP status code to be %s but was %s.'
                                 % (expected_status, actual_status))

    def start_chartmuseum(self, storage):
        self.stop_chartmuseum()
        os.chdir(self.rootdir)
        cmd = 'chartmuseum --debug --port=%d --storage="%s" ' % (common.PORT, storage)
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
        print(cmd)
        self.run_command(cmd, detach=True)

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
            tgz = glob.glob('*.tgz')[0]
            print('Uploading test chart package "%s"' % tgz)
            with open(tgz) as f:
                response = requests.post(url=charts_endpoint, data=f.read())
                print('POST %s' % charts_endpoint)
                print('HTTP STATUS: %s' % response.status_code)
                print('HTTP CONTENT: %s' % response.content)
                self.http_status_code_should_be(201, response.status_code)
            os.chdir('../')

    def upload_provenance_files(self):
        prov_endpoint = '%s/api/prov' % common.HELM_REPO_URL
        testcharts_dir = os.path.join(self.rootdir, common.TESTCHARTS_DIR)
        os.chdir(testcharts_dir)
        for d in os.listdir('.'):
            if not os.path.isdir(d):
                continue
            os.chdir(d)
            prov = glob.glob('*.tgz.prov')[0]
            print('Uploading provenance file "%s"' % prov)
            with open(prov) as f:
                response = requests.post(url=prov_endpoint, data=f.read())
                print('POST %s' % prov_endpoint)
                print('HTTP STATUS: %s' % response.status_code)
                print('HTTP CONTENT: %s' % response.content)
                self.http_status_code_should_be(201, response.status_code)
            os.chdir('../')

    def delete_test_charts(self):
        endpoint = '%s/api/charts' % common.HELM_REPO_URL
        testcharts_dir = os.path.join(self.rootdir, common.TESTCHARTS_DIR)
        os.chdir(testcharts_dir)
        for d in os.listdir('.'):
            if not os.path.isdir(d):
                continue
            os.chdir(d)
            tgz = glob.glob('*.tgz')[0]
            tmp = tgz[:-4].rsplit('-', 1)
            name = tmp[0]
            version = tmp[1]
            print('Delete test chart "%s-%s"' % (name, version))
            with open(tgz) as f:
                epoint = '%s/%s/%s' % (endpoint, name, version)
                response = requests.delete(url=epoint)
                print('HTTP STATUS: %s' % response.status_code)
                print('HTTP CONTENT: %s' % response.content)
                self.http_status_code_should_be(200, response.status_code)
            os.chdir('../')