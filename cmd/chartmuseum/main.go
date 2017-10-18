package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/chartmuseum/chartmuseum/pkg/chartmuseum"
	"github.com/chartmuseum/chartmuseum/pkg/storage"

	"github.com/urfave/cli"
)

var (
	crash = log.Fatal
	echo  = fmt.Print
	exit  = os.Exit

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
		Debug:                  c.Bool("debug"),
		LogJSON:                c.Bool("log-json"),
		EnableAPI:              !c.Bool("disable-api"),
		EnableMetrics:          !c.Bool("disable-metrics"),
		AllowOverwrite:         c.Bool("allow-overwrite"),
		ChartURL:               c.String("chart-url"),
		TlsCert:                c.String("tls-cert"),
		TlsKey:                 c.String("tls-key"),
		Username:               c.String("basic-auth-user"),
		Password:               c.String("basic-auth-pass"),
		StorageBackend:         backend,
		ChartPostFormFieldName: c.String("chart-post-form-field-name"),
		ProvPostFormFieldName:  c.String("prov-post-form-field-name"),
	}

	server, err := newServer(options)
	if err != nil {
		crash(err)
	}

	if c.Bool("gen-index") {
		echo(string(server.RepositoryIndex.Raw[:]))
		exit(0)
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
	))
}

func googleBackendFromContext(c *cli.Context) storage.Backend {
	crashIfContextMissingFlags(c, []string{"storage-google-bucket"})
	return storage.Backend(storage.NewGoogleCSBackend(
		c.String("storage-google-bucket"),
		c.String("storage-google-prefix"),
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
		EnvVar: "DEBUG",
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
}
