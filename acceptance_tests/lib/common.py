import os
import subprocess
import time

NOW = time.strftime('%Y%m%d%H%M%S')
PORT = 8080
HELM_REPO_NAME = 'chartmuseum'
HELM_REPO_URL = 'http://localhost:%d' % PORT
TESTCHARTS_DIR = 'testdata/charts'
TESTBADCHARTS_DIR = 'testdata/badcharts'
ACCEPTANCE_DIR = '.acceptance/'
STORAGE_DIR = os.path.join(ACCEPTANCE_DIR, 'storage/')
KEYRING = 'testdata/pgp/helm-test-key.pub'
LOGFILE = '.chartmuseum.log'
MAX_WAIT_SECONDS = 10

STORAGE_AMAZON_BUCKET = os.getenv('TEST_STORAGE_AMAZON_BUCKET')
STORAGE_AMAZON_REGION = os.getenv('TEST_STORAGE_AMAZON_REGION')
STORAGE_GOOGLE_BUCKET = os.getenv('TEST_STORAGE_GOOGLE_BUCKET')
STORAGE_MICROSOFT_CONTAINER = os.getenv('TEST_STORAGE_MICROSOFT_CONTAINER')
STORAGE_ALIBABA_BUCKET = os.getenv('TEST_STORAGE_ALIBABA_BUCKET')
STORAGE_ALIBABA_ENDPOINT = os.getenv('TEST_STORAGE_ALIBABA_ENDPOINT')
STORAGE_OPENSTACK_CONTAINER = os.getenv('TEST_STORAGE_OPENSTACK_CONTAINER')
STORAGE_OPENSTACK_REGION = os.getenv('TEST_STORAGE_OPENSTACK_REGION')
STORAGE_ORACLE_BUCKET = os.getenv('TEST_STORAGE_ORACLE_BUCKET')
STORAGE_ORACLE_COMPARTMENTID = os.getenv('TEST_STORAGE_ORACLE_COMPARTMENTID')
STORAGE_BAIDU_BUCKET = os.getenv('TEST_STORAGE_BAIDU_BUCKET')
STORAGE_BAIDU_ENDPOINT = os.getenv('TEST_STORAGE_BAIDU_ENDPOINT')

STORAGE_AMAZON_PREFIX = 'acceptance/%s' % NOW
STORAGE_GOOGLE_PREFIX = 'acceptance/%s' % NOW
STORAGE_MICROSOFT_PREFIX = 'acceptance/%s' % NOW
STORAGE_ALIBABA_PREFIX = 'acceptance/%s' % NOW
STORAGE_OPENSTACK_PREFIX = 'acceptance/%s' % NOW
STORAGE_ORACLE_PREFIX = 'acceptance/%s' % NOW
STORAGE_BAIDU_PREFIX = 'acceptance/%s' % NOW

class CommandRunner(object):
    def __init__(self):
        self.rc = 0
        self.pid = 0
        self.stdout = ''
        self.rootdir = os.path.realpath(os.path.join(__file__, '../../../'))

    def return_code_should_be(self, expected_rc):
        if int(expected_rc) != self.rc:
            raise AssertionError('Expected return code to be "%s" but was "%s".'
                                 % (expected_rc, self.rc))

    def return_code_should_not_be(self, expected_rc):
        if int(expected_rc) == self.rc:
            raise AssertionError('Expected return code not to be "%s".' % expected_rc)

    def output_contains(self, s):
        if s not in self.stdout:
            raise AssertionError('Output does not contain "%s".' % s)

    def output_does_not_contain(self, s):
        if s in self.stdout:
            raise AssertionError('Output contains "%s".' % s)

    def run_command(self, command, detach=False):
        process = subprocess.Popen(['/bin/bash', '-xc', command],
                                   stdout=subprocess.PIPE,
                                   stderr=subprocess.STDOUT)
        if not detach:
            stdout = process.communicate()[0].strip().decode()
            self.rc = process.returncode
            tmp = []
            for x in stdout.split('\n'):
                print(x)
                if not x.startswith('+ '): # Remove debug lines that start with "+ "
                    tmp.append(x)
            self.stdout = '\n'.join(tmp)
