import os
import subprocess
import time

NOW = time.strftime('%Y%m%d%H%M%S')
PORT = 8080
HELM_REPO_NAME = 'chartmuseum'
HELM_REPO_URL = 'http://localhost:%d' % PORT
TESTCHARTS_DIR = 'testdata/charts'
ACCEPTANCE_DIR = '.acceptance/'
STORAGE_DIR = os.path.join(ACCEPTANCE_DIR, 'storage/')
KEYRING = 'testdata/pgp/helm-test-key.pub'
LOGFILE = '.chartmuseum.log'

STORAGE_AMAZON_BUCKET = os.getenv('TEST_STORAGE_AMAZON_BUCKET')
STORAGE_AMAZON_REGION = os.getenv('TEST_STORAGE_AMAZON_REGION')
STORAGE_GOOGLE_BUCKET = os.getenv('TEST_STORAGE_GOOGLE_BUCKET')
STORAGE_MICROSOFT_CONTAINER = os.getenv('TEST_STORAGE_MICROSOFT_CONTAINER')
STORAGE_ALIBABA_BUCKET = os.getenv('TEST_STORAGE_ALIBABA_BUCKET')
STORAGE_ALIBABA_ENDPOINT = os.getenv('TEST_STORAGE_ALIBABA_ENDPOINT')

STORAGE_AMAZON_PREFIX = 'acceptance/%s' % NOW
STORAGE_GOOGLE_PREFIX = 'acceptance/%s' % NOW
STORAGE_MICROSOFT_PREFIX = 'acceptance/%s' % NOW
STORAGE_ALIBABA_PREFIX = 'acceptance/%s' % NOW


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
            stdout = process.communicate()[0].strip()
            print(stdout)
            self.rc = process.returncode
            # Remove debug lines that start with "+ "
            self.stdout = '\n'.join(filter(lambda x: not x.startswith('+ '), stdout.split('\n')))
