package router

import (
	"strings"

	"github.com/gin-gonic/gin"
)

/*
In order to allow params at the top level using gin (e.g. /:repo/...), this lengthy method is
unfortunately necessary. For more info, please see the "panic: wildcard route" error described
here: https://github.com/gin-gonic/gin/issues/388

This also adds the ability to accept a ":repo" param in the route containing a slash ("/"), so
that routes can be reused for different levels of multitenancy.

For example, the route GET /:repo/index.yaml will be matched differently depending on value used for --depth:

--depth=0:
	Path: "/index.yaml"
	Repo: ""
--depth=1:
	Path: "/myrepo/index.yaml"
	Repo: "myrepo"
--depth=2:
	Path: "/myorg/myrepo/index.yaml"
	Repo: "myorg/myrepo"
--depth=3:
	Path: "/myorg/myteam/myrepo/index.yaml"
	Repo: "myorg/myteam/myrepo"
*/
func match(routes []*Route, method string, url string, contextPath string, depth int) (*Route, []gin.Param) {
	var noRepoPathSplit []string
	var repo, repoPath, noRepoPath string
	var startIndex, numNoRepoPathParts int
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
		repoParts := pathSplit[startIndex : depth+startIndex]
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
