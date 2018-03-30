package router

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func (router *Router) matchRoute(c *gin.Context) {
	handle, params := match(router.Routes, c.Request.Method, c.Request.URL.Path, router.ContextPath, router.Depth)
	if handle == nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.Params = params
	handle(c)
}

func match(routes []*Route, method string, url string, contextPath string, depth int) (gin.HandlerFunc, []gin.Param) {
	var startIndex int
	var repo string
	var repoPath string
	var noRepoPath string

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
		repo = strings.Join(pathSplit[startIndex:depth+startIndex], "/")
		noRepoPath = "/" + strings.Join(pathSplit[depth+startIndex:], "/")
		repoPath = "/:repo" + noRepoPath
		if isApiRoute {
			repoPath = "/api" + repoPath
			noRepoPath = "/api" + noRepoPath
		}
	} else {
		noRepoPath = url
	}

	noRepoPathSplit := strings.Split(noRepoPath, "/")
	numNoRepoPathParts := len(noRepoPathSplit)

	for _, route := range routes {
		if route.Method != method {
			continue
		}
		if route.Path == url {
			return route.Handler, nil
		} else if route.Path == repoPath {
			return route.Handler, []gin.Param{{"repo", repo}}
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
					return route.Handler, params
				}
			}
		}
	}

	return nil, nil
}
