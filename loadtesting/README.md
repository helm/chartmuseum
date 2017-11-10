Loadtesting is made with the excellent Python [locust](https://locust.io/) library.

To facilitate installation, this loadtesting subproject uses pipenv.

Install pipenv

```
pip install pipenv
```

Install chartmuseum locust loadtesting

```
cd loadtesting
pipenv install
```

Start chartmuseum.

Start locust:

```
# run locust on a running chartmuseum instance
pipenv run locust --host http://localhost:8080
```

Open your locust console in your browser at http://localhost:8089, and start a new loadtest.
