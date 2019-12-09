module helm.sh/chartmuseum

go 1.12

require (
	github.com/alicebob/gopher-json v0.0.0-20180125190556-5a6b3ba71ee6 // indirect
	github.com/alicebob/miniredis v2.5.0+incompatible
	github.com/chartmuseum/auth v0.2.0
	github.com/chartmuseum/storage v0.5.0
	github.com/ghodss/yaml v1.0.0
	github.com/gin-contrib/size v0.0.0-20190528085907-355431950c57
	github.com/gin-gonic/gin v1.4.0
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/prometheus/client_golang v1.2.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	github.com/urfave/cli v1.20.0
	github.com/yuin/gopher-lua v0.0.0-20190514113301-1cd887cd7036 // indirect
	github.com/zsais/go-gin-prometheus v0.0.0-20181030200533-58963fb32f54
	go.uber.org/zap v1.10.0
	golang.org/x/sys v0.0.0-20191206220618-eeba5f6aabab // indirect
	helm.sh/helm/v3 v3.0.1
)

replace (
	github.com/Azure/go-autorest/autorest => github.com/Azure/go-autorest/autorest v0.9.0
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	github.com/miekg/dns => github.com/miekg/dns v0.0.0-20181005163659-0d29b283ac0f
	github.com/ugorji/go => github.com/ugorji/go v1.1.7
	gopkg.in/inf.v0 v0.9.1 => github.com/go-inf/inf v0.9.1
	gopkg.in/square/go-jose.v2 v2.3.0 => github.com/square/go-jose v2.3.0+incompatible
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.7
	rsc.io/letsencrypt => github.com/dmcgowan/letsencrypt v0.0.0-20160928181947-1847a81d2087
)
