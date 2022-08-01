from locust import HttpUser, TaskSet, task
import tarfile
import io

patch_version = 1

class UserBehavior(TaskSet):
    @task(10)
    def index(self):
        self.client.get("/index.yaml")

    @task(1)
    def post_new_chart(self):
        global patch_version
        # Create dummy 'chartmuseum-loadtest' chart package for which we only increment the patch version
        chart_post_field_name = 'chart'
        chart_name = 'chartmuseum-loadtest'
        chart_version = '0.0.%d' % patch_version
        patch_version += 1
        chart_fn = '%s-%s.tgz' % (chart_name, chart_version)

        tgz_buf = io.BytesIO()
        t = tarfile.open(mode = "w:gz", fileobj=tgz_buf)
        chart_content = b'name: %s\nversion: %s\n' % (chart_name.encode('utf8'), chart_version.encode('utf8'))
        tarinfo = tarfile.TarInfo('%s/Chart.yaml' % chart_name)
        tarinfo.size = len(chart_content)
        t.addfile(tarinfo=tarinfo, fileobj=io.BytesIO(chart_content))
        t.close()
        tgz_buf.seek(0)

        self.client.post('/api/charts', files={chart_post_field_name: (chart_fn, tgz_buf)})

        self.client.get(f'/api/charts/{chart_name}/{chart_version}')

class WebsiteUser(HttpUser):
    tasks = [UserBehavior]
    min_wait = 1000
    max_wait = 3000
