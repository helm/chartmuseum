import os

import common


class Helm(common.CommandRunner):
    def add_chart_repo(self):
        self.remove_chart_repo()
        self.run_command('helm repo add %s %s' % (common.HELM_REPO_NAME, common.HELM_REPO_URL))

    def remove_chart_repo(self):
        self.run_command('helm repo remove %s' % common.HELM_REPO_NAME)

    def search_for_chart(self, chart):
        self.run_command('helm search %s/%s' % (common.HELM_REPO_NAME, chart))

    def update_chart_repos(self):
        # "| head -n -1" prevents UnicodeDecodeError due to last line of output
        self.run_command('helm repo update | head -n -1 | \
                            grep "Successfully got an update from the \\"%s\\""' \
                         % common.HELM_REPO_NAME)

    def fetch_and_verify_chart(self, chart):
        os.chdir(self.rootdir)
        os.chdir(common.ACCEPTANCE_DIR)
        self.run_command('helm fetch --verify --keyring ../%s %s/%s' % (common.KEYRING, common.HELM_REPO_NAME, chart))
