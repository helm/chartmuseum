package router

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func (router *Router) matchRoute(c *gin.Context) {
	route, params := match(router.Routes, c.Request.Method, c.Request.URL.Path, router.ContextPath, router.Depth)
	if route == nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.Params = params

	if isRepoAction(route.Action) {
		authorized, responseHeaders := router.authorizeRequest(c.Request)
		for key, value := range responseHeaders {
			c.Header(key, value)
		}
		if !authorized {
			c.JSON(401, gin.H{"error": "unauthorized"})
			return
		}
	}

	route.Handler(c)
}

func match(routes []*Route, method string, url string, contextPath string, depth int) (*Route, []gin.Param) {
	var noRepoPathSplit []string
	var repo string
	var repoPath string
	var noRepoPath string
	var startIndex int
	var numNoRepoPathParts int
	var tryRepoRoutes bool

	if contextPath != "" {
		if url == contextPath {
			url = "/"
		} else if strings.HasPrefix(url, contextPath) {
			url = strings.Replace(url, contextPath, "", 1)
		} else {
			return nil, nil
		}
	}

	isApiRoute := strings.HasPrefix(url, "/api")
	if isApiRoute {
		startIndex = 2
	} else {
		startIndex = 1
	}

	pathSplit := strings.Split(url, "/")
	numParts := len(pathSplit)

	if numParts >= depth+startIndex {
		repoParts := pathSplit[startIndex:depth+startIndex]
		if len(repoParts) == depth {
			tryRepoRoutes = true
			repo = strings.Join(repoParts, "/")
			noRepoPath = "/" + strings.Join(pathSplit[depth+startIndex:], "/")
			repoPath = "/:repo" + noRepoPath
			if isApiRoute {
				repoPath = "/api" + repoPath
				noRepoPath = "/api" + noRepoPath
			}
			noRepoPathSplit = strings.Split(noRepoPath, "/")
			numNoRepoPathParts = len(noRepoPathSplit)
		}
	}

	for _, route := range routes {
		if route.Method != method {
			continue
		}
		if route.Path == url {
			return route, nil
		} else if tryRepoRoutes {
			if route.Path == repoPath {
				return route, []gin.Param{{"repo", repo}}
			} else {
				p := strings.Replace(route.Path, "/:repo", "", 1)
				if routeSplit := strings.Split(p, "/"); len(routeSplit) == numNoRepoPathParts {
					isMatch := true
					var params []gin.Param
					for i, part := range routeSplit {
						if paramSplit := strings.Split(part, ":"); len(paramSplit) > 1 {
							params = append(params, gin.Param{Key: paramSplit[1], Value: noRepoPathSplit[i]})
						} else if routeSplit[i] != noRepoPathSplit[i] {
							isMatch = false
							break
						}
					}
					if isMatch {
						params = append(params, gin.Param{Key: "repo", Value: repo})
						return route, params
					}
				}
			}
		}
	}

	return nil, nil
}
