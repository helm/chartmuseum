package multitenant

import (
	cm_storage "github.com/chartmuseum/storage"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	"helm.sh/helm/v3/pkg/repo"
	"time"
)

type (
	updateObject struct {
		RepoName     string             `json:"repo_name"`
		OpType       OperationType      `json:"operation_type"`
		ChartVersion *repo.ChartVersion `json:"chart_version"`
	}
)
type OperationType int

const (
	UpdateChart OperationType = 0
	AddChart    OperationType = 1
	DeleteChart OperationType = 2
)

var (
	updateObjectChan chan updateObject
)

func (server *MultiTenantServer) InitCacheTimer() {
	// consume
	InitUpdateObjectChan()
	go func() {
		server.Consumer()
	}()

	// delta update the cache every 5min (in case the files on the disk are manually manipulated)
	go func() {
		t := time.NewTicker(server.CacheInterval)
		for _ = range t.C {
			server.RebuildIndex()
		}
	}()
}

func InitUpdateObjectChan() {
	updateObjectChan = make(chan updateObject, 100)
	return
}

func (server *MultiTenantServer) Producer(repo string, operationType OperationType, chart *repo.ChartVersion) {
	uo := updateObject{
		RepoName:     repo,
		OpType:       operationType,
		ChartVersion: chart,
	}

	updateObjectChan <- uo
	log := server.Logger.ContextLoggingFn(&gin.Context{})
	log(cm_logger.InfoLevel, "send update", zap.Any("object", uo))
}

func (server *MultiTenantServer) Consumer() {
	server.Router.Logger.Info("start consumer")
	for {
		log := server.Logger.ContextLoggingFn(&gin.Context{})
		uo := <-updateObjectChan
		repo := uo.RepoName
		log(cm_logger.InfoLevel, "receive update", zap.String("repo", repo), zap.Any("uo", uo))
		tenant := server.Tenants[uo.RepoName]
		tenant.RegenerationLock.Lock()

		entry, err := server.initCacheEntry(log, repo)
		if err != nil {
			log(cm_logger.ErrorLevel, "initCacheEntry fail", zap.Error(err), zap.String("repo", repo))
			continue
		}

		index := entry.RepoIndex
		switch uo.OpType {
		case UpdateChart:
			index.UpdateEntry(uo.ChartVersion)
		case AddChart:
			index.AddEntry(uo.ChartVersion)
		case DeleteChart:
			index.RemoveEntry(uo.ChartVersion)
		default:
			log(cm_logger.ErrorLevel, "invalid operation type", zap.String("repo", repo),
				"operation_type", uo.OpType)
			continue
		}

		err = index.Regenerate()
		if err != nil {
			log(cm_logger.ErrorLevel, "regenerate fail", zap.Error(err), zap.String("repo", repo))
			continue
		}
		entry.RepoIndex = index

		err = server.saveCacheEntry(log, entry)
		if err != nil {
			log(cm_logger.ErrorLevel, "saveCacheEntry fail", zap.Error(err), zap.String("repo", repo))
			continue
		}

		if server.UseStatefiles {
			// Dont wait, save index-cache.yaml to storage in the background.
			// It is not crucial if this does not succeed, we will just log any errors
			go server.saveStatefile(log, uo.RepoName, entry.RepoIndex.Raw)
		}

		tenant.RegenerationLock.Unlock()
		log(cm_logger.InfoLevel, "success update", zap.String("repo", repo), zap.Any("uo", uo))
	}
}

func (server *MultiTenantServer) RebuildIndex() {
	for repo, _ := range server.Tenants {
		go func(repo string) {
			log := server.Logger.ContextLoggingFn(&gin.Context{})
			log(cm_logger.InfoLevel, "begin to rebuild index", zap.String("repo", repo))
			entry, err := server.initCacheEntry(log, repo)
			if err != nil {
				errStr := err.Error()
				log(cm_logger.ErrorLevel, errStr,
					"repo", repo,
				)
				return
			}

			fo := <-server.getChartList(log, repo)

			if fo.err != nil {
				errStr := fo.err.Error()
				log(cm_logger.ErrorLevel, errStr,
					"repo", repo,
				)
				return
			}

			objects := server.getRepoObjectSlice(entry)
			diff := cm_storage.GetObjectSliceDiff(objects, fo.objects, server.TimestampTolerance)

			// return fast if no changes
			if !diff.Change {
				log(cm_logger.DebugLevel, "No change detected between cache and storage",
					"repo", repo,
				)
				return
			}

			log(cm_logger.DebugLevel, "Change detected between cache and storage",
				"repo", repo,
			)

			ir := <-server.regenerateRepositoryIndex(log, entry, diff)
			if ir.err != nil {
				errStr := ir.err.Error()
				log(cm_logger.ErrorLevel, errStr,
					"repo", repo,
				)
				return
			}
			entry.RepoIndex = ir.index

			if server.UseStatefiles {
				// Dont wait, save index-cache.yaml to storage in the background.
				// It is not crucial if this does not succeed, we will just log any errors
				go server.saveStatefile(log, repo, ir.index.Raw)
			}

		}(repo)
	}
}
