package events

import (
	"github.com/gin-gonic/gin"
	helm_repo "helm.sh/helm/v3/pkg/repo"
)

type OperationType int

type Event struct {
	Context      *gin.Context            `json:"-"`
	RepoName     string                  `json:"repo_name"`
	OpType       OperationType           `json:"operation_type"`
	ChartVersion *helm_repo.ChartVersion `json:"chart_version"`
}

const (
	UpdateChart OperationType = 0
	AddChart    OperationType = 1
	DeleteChart OperationType = 2
)
