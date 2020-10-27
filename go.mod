module helm.sh/chartmuseum

go 1.15

replace (
	github.com/NetEase-Object-Storage/nos-golang-sdk => github.com/karuppiah7890/nos-golang-sdk v0.0.0-20191116042345-0792ba35abcc
	go.etcd.io/etcd => github.com/eddycjy/etcd v0.5.0-alpha.5.0.20200218102753-4258cdd2efdf
	github.com/chartmuseum/auth => github.com/marcoklaassen/auth v0.5.0
)

require (
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/alicebob/miniredis v2.5.0+incompatible
	github.com/chartmuseum/auth v0.4.2
	github.com/chartmuseum/storage v0.9.1
	github.com/ghodss/yaml v1.0.0
	github.com/gin-contrib/size v0.0.0-20200815104238-dc717522c4e2
	github.com/gin-gonic/gin v1.6.3
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/gomodule/redigo v1.8.2 // indirect
	github.com/prometheus/client_golang v1.7.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/urfave/cli v1.22.4
	github.com/yuin/gopher-lua v0.0.0-20200816102855-ee81675732da // indirect
	github.com/zsais/go-gin-prometheus v0.1.0
	go.uber.org/zap v1.16.0
	helm.sh/helm/v3 v3.3.1
)
