package router

import (
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"

	"github.com/gin-gonic/gin"
)

type (
	// RouterGroups TODO
	RouterGroups struct {
		ReadAccess  *gin.RouterGroup
		WriteAccess *gin.RouterGroup
		SysInfo     *gin.RouterGroup
	}

	// RouterGroupsOptions TODO
	RouterGroupsOptions struct {
		Logger       *cm_logger.Logger
		Router       *Router
		Username     string
		Password     string
		ContextPath  string
		AnonymousGet bool
	}
)

// NewRouterGroups creates a new RouterGroups instance
func NewRouterGroups(options RouterGroupsOptions) *RouterGroups {
	sysInfoGroup := &options.Router.RouterGroup
	readAccessGroup := &options.Router.RouterGroup
	writeAccessGroup := &options.Router.RouterGroup

	// Reconfigure read-access, write-access groups if basic auth is enabled
	if options.Username != "" && options.Password != "" {
		basicAuthGroup := options.Router.Group("")
		users := make(map[string]string)
		users[options.Username] = options.Password
		basicAuthGroup.Use(gin.BasicAuthForRealm(users, "ChartMuseum"))
		writeAccessGroup = basicAuthGroup
		if options.AnonymousGet {
			options.Logger.Debug("Anonymous GET enabled")
		} else {
			readAccessGroup = basicAuthGroup
		}
	}

	routerGroups := &RouterGroups{
		ReadAccess:  readAccessGroup,
		WriteAccess: writeAccessGroup,
		SysInfo:     sysInfoGroup,
	}

	return routerGroups
}
