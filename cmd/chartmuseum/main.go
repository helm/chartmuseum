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
	crash     = log.Fatal
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
		Debug:          c.Bool("debug"),
		LogJSON:        c.Bool("log-json"),
		StorageBackend: backend,
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
	crashIfContextMissingFlags(c, []string{"storage-amazon-bucket", "storage-amazon-region"})
	return storage.Backend(storage.NewAmazonS3Backend(
		c.String("storage-amazon-bucket"),
		c.String("storage-amazon-prefix"),
		c.String("storage-amazon-region"),
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
		Name:   "debug",
		Usage:  "show debug messages",
		EnvVar: "DEBUG",
	},
	cli.BoolFlag{
		Name:   "log-json",
		Usage:  "output structured logs as json",
		EnvVar: "LOG_JSON",
	},
	cli.IntFlag{
		Name:   "port",
		Value:  8080,
		Usage:  "port to listen on",
		EnvVar: "PORT",
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
		Value:  "",
		Usage:  "prefix to store charts for --storage-amazon-bucket",
		EnvVar: "STORAGE_AMAZON_PREFIX",
	},
	cli.StringFlag{
		Name:   "storage-amazon-region",
		Usage:  "region of --storage-amazon-bucket",
		EnvVar: "STORAGE_AMAZON_REGION",
	},
	cli.StringFlag{
		Name:   "storage-google-bucket",
		Usage:  "gcs bucket to store charts for google storage backend",
		EnvVar: "STORAGE_GOOGLE_BUCKET",
	},
	cli.StringFlag{
		Name:   "storage-google-prefix",
		Value:  "",
		Usage:  "prefix to store charts for --storage-google-bucket",
		EnvVar: "STORAGE_GOOGLE_PREFIX",
	},
}
