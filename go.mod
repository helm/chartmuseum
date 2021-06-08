module helm.sh/chartmuseum

go 1.16

replace (
	github.com/NetEase-Object-Storage/nos-golang-sdk => github.com/karuppiah7890/nos-golang-sdk v0.0.0-20191116042345-0792ba35abcc
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	go.etcd.io/etcd => github.com/eddycjy/etcd v0.5.0-alpha.5.0.20200218102753-4258cdd2efdf
	google.golang.org/grpc => google.golang.org/grpc v1.29.1
)

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/alicebob/miniredis v2.5.0+incompatible
	github.com/chartmuseum/auth v0.4.5
	github.com/chartmuseum/storage v0.10.5
	github.com/ghodss/yaml v1.0.0
	github.com/gin-contrib/size v0.0.0-20200916080119-37b334d93b20
	github.com/gin-gonic/gin v1.6.3
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/gomodule/redigo v1.8.4 // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli v1.22.5
	github.com/yuin/gopher-lua v0.0.0-20200816102855-ee81675732da // indirect
	github.com/zsais/go-gin-prometheus v0.1.0
	go.uber.org/zap v1.16.0
	helm.sh/helm/v3 v3.5.1
)
