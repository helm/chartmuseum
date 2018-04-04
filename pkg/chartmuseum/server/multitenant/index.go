package multitenant

import (
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "github.com/kubernetes-helm/chartmuseum/pkg/repo"
	cm_storage "github.com/kubernetes-helm/chartmuseum/pkg/storage"
)

var (
	indexFileContentType = "application/x-yaml"
)

func (server *MultiTenantServer) getIndexFile(log cm_logger.LoggingFn, repo string) (*cm_repo.Index, *HTTPError) {
	server.initCachedIndexFile(log, repo)

	fo := <-server.getChartList(log, repo)

	if fo.err != nil {
		errStr := fo.err.Error()
		log(cm_logger.ErrorLevel, errStr,
			"repo", repo,
		)
		return nil, &HTTPError{500, errStr}
	}

	diff := cm_storage.GetObjectSliceDiff(server.IndexCache[repo].StorageCache, fo.objects)

	// return fast if no changes
	if !diff.Change {
		return server.IndexCache[repo].RepositoryIndex, nil
	}

	ir := <-server.regenerateRepositoryIndex(log, repo, diff, fo.objects)

	if ir.err != nil {
		errStr := ir.err.Error()
		log(cm_logger.ErrorLevel, errStr,
			"repo", repo,
		)
		return ir.index, &HTTPError{500, errStr}
	}

	return ir.index, nil
}
