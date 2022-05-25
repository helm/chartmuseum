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

package multitenant

import (
	"net/http"

	"github.com/ghodss/yaml"

	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "helm.sh/chartmuseum/pkg/repo"
)

const artifactHubFileContentType = "application/x-yaml"

func (server *MultiTenantServer) getArtifactHubYml(log cm_logger.LoggingFn, repo string) ([]byte, *HTTPError) {
	if _, ok := server.ArtifactHubRepoID[repo]; !ok {
		return nil, &HTTPError{http.StatusNotFound, "Artifact Hub repository ID not found"}
	}
	artifactHubFile := &cm_repo.ArtifactHubFile{
		RepoID: server.ArtifactHubRepoID[repo],
	}
	log(cm_logger.DebugLevel, "Generating artifacthub-repo.yml file", "repo", repo)
	rawArtifactHubFile, err := yaml.Marshal(&artifactHubFile)
	if err != nil {
		errStr := "failed to generate artifacthub-repo.yml file"
		log(cm_logger.ErrorLevel, errStr,
			"repo", repo,
		)
		return nil, &HTTPError{http.StatusInternalServerError, errStr}
	}
	return rawArtifactHubFile, nil
}
