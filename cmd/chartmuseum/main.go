/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/chartmuseum/storage"

	"helm.sh/chartmuseum/pkg/cache"
	"helm.sh/chartmuseum/pkg/chartmuseum"
	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	"helm.sh/chartmuseum/pkg/config"

	"github.com/urfave/cli"
)

var (
	crash = log.Fatal

	newServer = chartmuseum.NewServer

	// Version is the semantic version (added at compile time)
	Version string

	// Revision is the git commit id (added at compile time)
	Revision string
)

func main() {
	app := cli.NewApp()
	app.Name = "ChartMuseum"
	app.Version = fmt.Sprintf("%s (build %s)", Version, Revision)
	app.Usage = "Helm Chart Repository with support for Amazon S3, Google Cloud Storage, Oracle Cloud Infrastructure Object Storage and Openstack"
	app.Action = cliHandler
	app.Flags = config.CLIFlags
	app.Run(os.Args)
}

func cliHandler(c *cli.Context) {
	conf := config.NewConfig()
	err := conf.UpdateFromCLIContext(c)
	if err != nil {
		crash(err)
	}

	logger, err := cm_logger.NewLogger(cm_logger.LoggerOptions{
		Debug:   conf.GetBool("debug"),
		LogJSON: conf.GetBool("logjson"),
	})
	if err != nil {
		crash(err)
	}

	conf.ShowDeprecationWarnings(c, logger)

	backend := backendFromConfig(conf)
	store := storeFromConfig(conf)

	options := chartmuseum.ServerOptions{
		Version:                Version,
		StorageBackend:         backend,
		ExternalCacheStore:     store,
		Logger:                 logger,
		TimestampTolerance:     conf.GetDuration("storage.timestamptolerance"),
		ChartURL:               conf.GetString("charturl"),
		TlsCert:                conf.GetString("tls.cert"),
		TlsKey:                 conf.GetString("tls.key"),
		TlsCACert:              conf.GetString("tls.cacert"),
		Username:               conf.GetString("basicauth.user"),
		Password:               conf.GetString("basicauth.pass"),
		ChartPostFormFieldName: conf.GetString("chartpostformfieldname"),
		ProvPostFormFieldName:  conf.GetString("provpostformfieldname"),
		ContextPath:            conf.GetString("contextpath"),
		LogHealth:              conf.GetBool("loghealth"),
		LogLatencyInteger:      conf.GetBool("loglatencyinteger"),
		EnableAPI:              !conf.GetBool("disableapi"),
		DisableDelete:          conf.GetBool("disabledelete"),
		UseStatefiles:          !conf.GetBool("disablestatefiles"),
		AllowOverwrite:         conf.GetBool("allowoverwrite"),
		AllowForceOverwrite:    !conf.GetBool("disableforceoverwrite"),
		EnableMetrics:          conf.GetBool("enablemetrics"),
		AnonymousGet:           conf.GetBool("authanonymousget"),
		GenIndex:               conf.GetBool("genindex"),
		MaxStorageObjects:      conf.GetInt("maxstorageobjects"),
		IndexLimit:             conf.GetInt("indexlimit"),
		Depth:                  conf.GetInt("depth"),
		MaxUploadSize:          conf.GetInt("maxuploadsize"),
		BearerAuth:             conf.GetBool("bearerauth"),
		AuthRealm:              conf.GetString("authrealm"),
		AuthService:            conf.GetString("authservice"),
		AuthCertPath:           conf.GetString("authcertpath"),
		AuthActionsSearchPath:  conf.GetString("authactionssearchpath"),
		DepthDynamic:           conf.GetBool("depthdynamic"),
		CORSAllowOrigin:        conf.GetString("cors.alloworigin"),
		WriteTimeout:           conf.GetInt("writetimeout"),
		ReadTimeout:            conf.GetInt("readtimeout"),
		EnforceSemver2:         conf.GetBool("enforce-semver2"),
		CacheInterval:          conf.GetDuration("cacheinterval"),
		Host:                   conf.GetString("listen.host"),
		PerChartLimit:          conf.GetInt("per-chart-limit"),
		WebTemplatePath:        conf.GetString("web-template-path"),
		ArtifactHubRepoID:      conf.GetStringMapString("artifact-hub-repo-id"),
		AlwaysRegenerateIndex:  conf.GetBool("always-regenerate-chart-index"),
	}

	server, err := newServer(options)
	if err != nil {
		crash(err)
	}

	server.Listen(conf.GetInt("port"))
}

func backendFromConfig(conf *config.Config) storage.Backend {
	crashIfConfigMissingVars(conf, []string{"storage.backend"})

	var backend storage.Backend

	storageFlag := strings.ToLower(conf.GetString("storage.backend"))
	switch storageFlag {
	case "local":
		backend = localBackendFromConfig(conf)
	case "amazon":
		backend = amazonBackendFromConfig(conf)
	case "google":
		backend = googleBackendFromConfig(conf)
	case "oracle":
		backend = oracleBackendFromConfig(conf)
	case "microsoft":
		backend = microsoftBackendFromConfig(conf)
	case "alibaba":
		backend = alibabaBackendFromConfig(conf)
	case "openstack":
		backend = openstackBackendFromConfig(conf)
	case "baidu":
		backend = baiduBackendFromConfig(conf)
	case "etcd":
		backend = etcdBackendFromConfig(conf)
	case "tencent":
		backend = tencentBackendFromConfig(conf)
	default:
		crash("Unsupported storage backend: ", storageFlag)
	}

	return backend
}

func localBackendFromConfig(conf *config.Config) storage.Backend {
	crashIfConfigMissingVars(conf, []string{"storage.local.rootdir"})
	return storage.NewLocalFilesystemBackend(
		conf.GetString("storage.local.rootdir"),
	)
}

func amazonBackendFromConfig(conf *config.Config) storage.Backend {
	// If using alternative s3 endpoint (e.g. Minio) default region to us-east-1
	if conf.GetString("storage.amazon.endpoint") != "" && conf.GetString("storage.amazon.region") == "" {
		conf.Set("storage.amazon.region", "us-east-1")
	}
	crashIfConfigMissingVars(conf, []string{"storage.amazon.bucket", "storage.amazon.region"})
	return storage.NewAmazonS3Backend(
		conf.GetString("storage.amazon.bucket"),
		conf.GetString("storage.amazon.prefix"),
		conf.GetString("storage.amazon.region"),
		conf.GetString("storage.amazon.endpoint"),
		conf.GetString("storage.amazon.sse"),
	)
}

func googleBackendFromConfig(conf *config.Config) storage.Backend {
	crashIfConfigMissingVars(conf, []string{"storage.google.bucket"})
	return storage.NewGoogleCSBackend(
		conf.GetString("storage.google.bucket"),
		conf.GetString("storage.google.prefix"),
	)
}

func oracleBackendFromConfig(conf *config.Config) storage.Backend {
	crashIfConfigMissingVars(conf, []string{"storage.oracle.bucket", "storage.oracle.compartmentid"})
	return storage.NewOracleCSBackend(
		conf.GetString("storage.oracle.bucket"),
		conf.GetString("storage.oracle.prefix"),
		conf.GetString("storage.oracle.region"),
		conf.GetString("storage.oracle.compartmentid"),
	)
}

func microsoftBackendFromConfig(conf *config.Config) storage.Backend {
	crashIfConfigMissingVars(conf, []string{"storage.microsoft.container"})
	return storage.NewMicrosoftBlobBackend(
		conf.GetString("storage.microsoft.container"),
		conf.GetString("storage.microsoft.prefix"),
	)
}

func alibabaBackendFromConfig(conf *config.Config) storage.Backend {
	crashIfConfigMissingVars(conf, []string{"storage.alibaba.bucket"})
	return storage.NewAlibabaCloudOSSBackend(
		conf.GetString("storage.alibaba.bucket"),
		conf.GetString("storage.alibaba.prefix"),
		conf.GetString("storage.alibaba.endpoint"),
		conf.GetString("storage.alibaba.sse"),
	)
}

func openstackBackendFromConfig(conf *config.Config) storage.Backend {
	var backend storage.Backend
	switch conf.GetString("storage.openstack.auth") {
	case "v1":
		crashIfConfigMissingVars(conf, []string{"storage.openstack.container"})
		backend = storage.NewOpenstackOSBackendV1Auth(
			conf.GetString("storage.openstack.container"),
			conf.GetString("storage.openstack.prefix"),
			conf.GetString("storage.openstack.cacert"),
		)
	case "auto":
		crashIfConfigMissingVars(conf, []string{"storage.openstack.container", "storage.openstack.region"})
		backend = storage.NewOpenstackOSBackend(
			conf.GetString("storage.openstack.container"),
			conf.GetString("storage.openstack.prefix"),
			conf.GetString("storage.openstack.region"),
			conf.GetString("storage.openstack.cacert"),
		)
	default:
		crash("Unsupported OpenStack auth protocol: ", conf.GetString("storage.openstack.auth"))
	}
	return backend
}

func baiduBackendFromConfig(conf *config.Config) storage.Backend {
	crashIfConfigMissingVars(conf, []string{"storage.baidu.bucket"})
	return storage.NewBaiDuBOSBackend(
		conf.GetString("storage.baidu.bucket"),
		conf.GetString("storage.baidu.prefix"),
		conf.GetString("storage.baidu.endpoint"),
	)
}

func etcdBackendFromConfig(conf *config.Config) storage.Backend {
	crashIfConfigMissingVars(conf, []string{"storage.etcd.cafile",
		"storage.etcd.certfile",
		"storage.etcd.keyfile",
		"storage.etcd.prefix"})
	return storage.NewEtcdCSBackend(
		conf.GetString("storage.etcd.endpoint"),
		conf.GetString("storage.etcd.cafile"),
		conf.GetString("storage.etcd.certfile"),
		conf.GetString("storage.etcd.keyfile"),
		conf.GetString("storage.etcd.prefix"),
	)
}

func tencentBackendFromConfig(conf *config.Config) storage.Backend {
	crashIfConfigMissingVars(conf, []string{"storage.tencent.bucket"})
	return storage.NewTencentCloudCOSBackend(
		conf.GetString("storage.tencent.bucket"),
		conf.GetString("storage.tencent.prefix"),
		conf.GetString("storage.tencent.endpoint"),
	)
}

func storeFromConfig(conf *config.Config) cache.Store {
	if conf.GetString("cache.store") == "" {
		return nil
	}

	var store cache.Store

	cacheFlag := strings.ToLower(conf.GetString("cache.store"))
	switch cacheFlag {
	case "redis":
		store = redisCacheFromConfig(conf)
	default:
		crash("Unsupported cache store: ", cacheFlag)
	}

	return store
}

func redisCacheFromConfig(conf *config.Config) cache.Store {
	crashIfConfigMissingVars(conf, []string{"cache.redis.addr"})
	return cache.Store(cache.NewRedisStore(
		conf.GetString("cache.redis.addr"),
		conf.GetString("cache.redis.password"),
		conf.GetInt("cache.redis.db"),
	))
}

func crashIfConfigMissingVars(conf *config.Config, vars []string) {
	var missing []string
	for _, v := range vars {
		if conf.GetString(v) == "" {
			flag := config.GetCLIFlagFromVarName(v)
			missing = append(missing, fmt.Sprintf("--%s", flag))
		}
	}
	if len(missing) > 0 {
		crash("Missing required flags(s): ", strings.Join(missing, ", "))
	}
}
