set -xeu
for i in {1..2000}
do
  curl --fail http://localhost:8080/api/charts/chartmuseum-loadtest/0.0.${i}
done
