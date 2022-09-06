set -xeu

num_of_charts=$(curl localhost:8080/index.yaml | yq '.entries.chartmuseum-loadtest[0].version' | cut -d '.' -f3)
for i in $(seq 1 $num_of_charts)
do
  curl --fail http://localhost:8080/api/charts/chartmuseum-loadtest/0.0.${i}
done
