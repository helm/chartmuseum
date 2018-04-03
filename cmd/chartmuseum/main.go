package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"

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
	app.Usage = "Helm Chart Repository with support for Amazon S3 and Google Cloud Storage"
	app.Action = cliHandler
	app.Flags = cliFlags
	app.Run(os.Args)
}

func cliHandler(c *cli.Context) {
	backend := backendFromContext(c)

	options := chartmuseum.ServerOptions{
		StorageBackend:         backend,
		ChartURL:               c.String("chart-url"),
		TlsCert:                c.String("tls-cert"),
		TlsKey:                 c.String("tls-key"),
		Username:               c.String("basic-auth-user"),
		Password:               c.String("basic-auth-pass"),
		ChartPostFormFieldName: c.String("chart-post-form-field-name"),
		ProvPostFormFieldName:  c.String("prov-post-form-field-name"),
		ContextPath:            c.String("context-path"),
		LogJSON:                c.Bool("log-json"),
		Debug:                  c.Bool("debug"),
		EnableAPI:              !c.Bool("disable-api"),
		AllowOverwrite:         c.Bool("allow-overwrite"),
		EnableMetrics:          !c.Bool("disable-metrics"),
		AnonymousGet:           c.Bool("auth-anonymous-get"),
		GenIndex:               c.Bool("gen-index"),
		IndexLimit:             c.Int("index-limit"),
		Depth:                  c.Int("depth"),
	}

	server, err := newServer(options)
	if err != nil {
		crash(err)
	}

	server.Listen(c.Int("port"))
}

func backendFromContext(c *cli.Context) storage.Backend {
	crashIfContextMissingFlags(c, []string{"storage"})

	var backend storage.Backend

	storageFlag := strings.ToLower(c.String("storage"))
	switch storageFlag {
	case "local":
		backend = localBackendFromContext(c)
	case "amazon":
		backend = amazonBackendFromContext(c)
	case "google":
		backend = googleBackendFromContext(c)
	case "microsoft":
		backend = microsoftBackendFromContext(c)
	case "alibaba":
		backend = alibabaBackendFromContext(c)
	default:
		crash("Unsupported storage backend: ", storageFlag)
	}

	return backend
}

func localBackendFromContext(c *cli.Context) storage.Backend {
	crashIfContextMissingFlags(c, []string{"storage-local-rootdir"})
	return storage.Backend(storage.NewLocalFilesystemBackend(
		c.String("storage-local-rootdir"),
	))
}

func amazonBackendFromContext(c *cli.Context) storage.Backend {
	// If using alternative s3 endpoint (e.g. Minio) default region to us-east-1
	if c.String("storage-amazon-endpoint") != "" && c.String("storage-amazon-region") == "" {
		c.Set("storage-amazon-region", "us-east-1")
	}
	crashIfContextMissingFlags(c, []string{"storage-amazon-bucket", "storage-amazon-region"})
	return storage.Backend(storage.NewAmazonS3Backend(
		c.String("storage-amazon-bucket"),
		c.String("storage-amazon-prefix"),
		c.String("storage-amazon-region"),
		c.String("storage-amazon-endpoint"),
		c.String("storage-amazon-sse"),
	))
}

func googleBackendFromContext(c *cli.Context) storage.Backend {
	crashIfContextMissingFlags(c, []string{"storage-google-bucket"})
	return storage.Backend(storage.NewGoogleCSBackend(
		c.String("storage-google-bucket"),
		c.String("storage-google-prefix"),
	))
}

func microsoftBackendFromContext(c *cli.Context) storage.Backend {
	crashIfContextMissingFlags(c, []string{"storage-microsoft-container"})
	return storage.Backend(storage.NewMicrosoftBlobBackend(
		c.String("storage-microsoft-container"),
		c.String("storage-microsoft-prefix"),
	))
}

func alibabaBackendFromContext(c *cli.Context) storage.Backend {
	crashIfContextMissingFlags(c, []string{"storage-alibaba-bucket"})
	return storage.Backend(storage.NewAlibabaCloudOSSBackend(
		c.String("storage-alibaba-bucket"),
		c.String("storage-alibaba-prefix"),
		c.String("storage-alibaba-endpoint"),
		c.String("storage-alibaba-sse"),
	))
}

func crashIfContextMissingFlags(c *cli.Context, flags []string) {
	missing := []string{}
	for _, flag := range flags {
		if c.String(flag) == "" {
			missing = append(missing, fmt.Sprintf("--%s", flag))
		}
	}
	if len(missing) > 0 {
		crash("Missing required flags(s): ", strings.Join(missing, ", "))
	}
}

var cliFlags = []cli.Flag{
	cli.BoolFlag{
		Name:   "gen-index",
		Usage:  "generate index.yaml, print to stdout and exit",
		EnvVar: "GEN_INDEX",
	},
	cli.BoolFlag{
		Name:   "debug",
		Usage:  "show debug messages",
		EnvVar: "DEBUG",
	},
	cli.BoolFlag{
		Name:   "log-json",
		Usage:  "output structured logs as json",
		EnvVar: "LOG_JSON",
	},
	cli.BoolFlag{
		Name:   "disable-metrics",
		Usage:  "disable Prometheus metrics",
		EnvVar: "DISABLE_METRICS",
	},
	cli.BoolFlag{
		Name:   "disable-api",
		Usage:  "disable all routes prefixed with /api",
		EnvVar: "DISABLE_API",
	},
	cli.BoolFlag{
		Name:   "allow-overwrite",
		Usage:  "allow chart versions to be re-uploaded",
		EnvVar: "ALLOW_OVERWRITE",
	},
	cli.IntFlag{
		Name:   "port",
		Value:  8080,
		Usage:  "port to listen on",
		EnvVar: "PORT",
	},
	cli.StringFlag{
		Name:   "chart-url",
		Usage:  "absolute url for .tgzs in index.yaml",
		EnvVar: "CHART_URL",
	},
	cli.StringFlag{
		Name:   "basic-auth-user",
		Usage:  "username for basic http authentication",
		EnvVar: "BASIC_AUTH_USER",
	},
	cli.StringFlag{
		Name:   "basic-auth-pass",
		Usage:  "password for basic http authentication",
		EnvVar: "BASIC_AUTH_PASS",
	},
	cli.BoolFlag{
		Name:   "auth-anonymous-get",
		Usage:  "allow anonymous GET operations when auth is used",
		EnvVar: "AUTH_ANONYMOUS_GET",
	},
	cli.StringFlag{
		Name:   "tls-cert",
		Usage:  "path to tls certificate chain file",
		EnvVar: "TLS_CERT",
	},
	cli.StringFlag{
		Name:   "tls-key",
		Usage:  "path to tls key file",
		EnvVar: "TLS_KEY",
	},
	cli.StringFlag{
		Name:   "storage",
		Usage:  "storage backend, can be one of: local, amazon, google",
		EnvVar: "STORAGE",
	},
	cli.StringFlag{
		Name:   "storage-local-rootdir",
		Usage:  "directory to store charts for local storage backend",
		EnvVar: "STORAGE_LOCAL_ROOTDIR",
	},
	cli.StringFlag{
		Name:   "storage-amazon-bucket",
		Usage:  "s3 bucket to store charts for amazon storage backend",
		EnvVar: "STORAGE_AMAZON_BUCKET",
	},
	cli.StringFlag{
		Name:   "storage-amazon-prefix",
		Usage:  "prefix to store charts for --storage-amazon-bucket",
		EnvVar: "STORAGE_AMAZON_PREFIX",
	},
	cli.StringFlag{
		Name:   "storage-amazon-region",
		Usage:  "region of --storage-amazon-bucket",
		EnvVar: "STORAGE_AMAZON_REGION",
	},
	cli.StringFlag{
		Name:   "storage-amazon-endpoint",
		Usage:  "alternative s3 endpoint",
		EnvVar: "STORAGE_AMAZON_ENDPOINT",
	},
	cli.StringFlag{
		Name:   "storage-amazon-sse",
		Usage:  "server side encryption algorithm",
		EnvVar: "STORAGE_AMAZON_SSE",
	},
	cli.StringFlag{
		Name:   "storage-google-bucket",
		Usage:  "gcs bucket to store charts for google storage backend",
		EnvVar: "STORAGE_GOOGLE_BUCKET",
	},
	cli.StringFlag{
		Name:   "storage-google-prefix",
		Usage:  "prefix to store charts for --storage-google-bucket",
		EnvVar: "STORAGE_GOOGLE_PREFIX",
	},
	cli.StringFlag{
		Name:   "storage-microsoft-container",
		Usage:  "container to store charts for microsoft storage backend",
		EnvVar: "STORAGE_MICROSOFT_CONTAINER",
	},
	cli.StringFlag{
		Name:   "storage-microsoft-prefix",
		Usage:  "prefix to store charts for --storage-microsoft-prefix",
		EnvVar: "STORAGE_MICROSOFT_PREFIX",
	},
	cli.StringFlag{
		Name:   "storage-alibaba-bucket",
		Usage:  "OSS bucket to store charts for Alibaba Cloud storage backend",
		EnvVar: "STORAGE_ALIBABA_BUCKET",
	},
	cli.StringFlag{
		Name:   "storage-alibaba-prefix",
		Usage:  "prefix to store charts for --storage-alibaba-cloud-bucket",
		EnvVar: "STORAGE_ALIBABA_PREFIX",
	},
	cli.StringFlag{
		Name:   "storage-alibaba-endpoint",
		Usage:  "OSS endpoint",
		EnvVar: "STORAGE_ALIBABA_ENDPOINT",
	},
	cli.StringFlag{
		Name:   "storage-alibaba-sse",
		Usage:  "server side encryption algorithm for Alibaba Cloud storage backend, AES256 or KMS",
		EnvVar: "STORAGE_ALIBABA_SSE",
	},
	cli.StringFlag{
		Name:   "chart-post-form-field-name",
		Value:  "chart",
		Usage:  "form field which will be queried for the chart file content",
		EnvVar: "CHART_POST_FORM_FIELD_NAME",
	},
	cli.StringFlag{
		Name:   "prov-post-form-field-name",
		Value:  "prov",
		Usage:  "form field which will be queried for the provenance file content",
		EnvVar: "PROV_POST_FORM_FIELD_NAME",
	},
	cli.IntFlag{
		Name:   "index-limit",
		Value:  0,
		Usage:  "parallel scan limit for the repo indexer",
		EnvVar: "INDEX_LIMIT",
	},
	cli.StringFlag{
		Name:   "context-path",
		Value:  "",
		Usage:  "base context path",
		EnvVar: "CONTEXT_PATH",
	},
	cli.IntFlag{
		Name:   "depth",
		Value:  0,
		Usage:  "levels of nested repos for multitenancy",
		EnvVar: "DEPTH",
	},
}
