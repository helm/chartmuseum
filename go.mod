module helm.sh/chartmuseum

go 1.13

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible
	github.com/NetEase-Object-Storage/nos-golang-sdk => github.com/karuppiah7890/nos-golang-sdk v0.0.0-20191116042345-0792ba35abcc
	go.etcd.io/etcd => github.com/eddycjy/etcd v0.5.0-alpha.5.0.20200218102753-4258cdd2efdf
)

require (
	github.com/alicebob/gopher-json v0.0.0-20180125190556-5a6b3ba71ee6 // indirect
	github.com/alicebob/miniredis v2.5.0+incompatible
	github.com/chartmuseum/auth v0.4.0
	github.com/chartmuseum/storage v0.7.0
	github.com/ghodss/yaml v1.0.0
	github.com/gin-contrib/size v0.0.0-20191128031627-745aacce0004
	github.com/gin-gonic/gin v1.5.0
	github.com/go-redis/redis v6.15.7+incompatible
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/prometheus/client_golang v1.0.0
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.5.1
	github.com/urfave/cli v1.20.0
	github.com/yuin/gopher-lua v0.0.0-20191220021717-ab39c6098bdb // indirect
	github.com/zsais/go-gin-prometheus v0.1.0
	go.uber.org/zap v1.10.0
	helm.sh/helm/v3 v3.1.1
)
