package router

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

func (router *Router) matchRoute(c *gin.Context) {
	handle, params := match(router.Routes, c.Request.URL.Path, router.ContextPath, router.Depth)
	if handle == nil {
		c.AbortWithStatus(404)
		return
	}
	c.Params = params
	handle(c)
}

func match(routes []*Route, url string, contextPath string, depth int) (gin.HandlerFunc, []gin.Param) {
	pathSplit := strings.Split(url, "/")
	repo := strings.Join(pathSplit[1:depth+1], "/")
	pathPart := "/" + strings.Join(pathSplit[depth+1:], "/")
	repoPath := "/:repo" + pathPart

	fmt.Println(fmt.Sprintf("pathPart: %s, repoPath: %s, repo: %s", pathPart, repoPath, repo))

	for _, route := range routes {
		if route.Path == pathPart {

			fmt.Println("MATCH", url, route.Path)
			return route.Handler, nil //params
		} else if route.Path == repoPath {

			fmt.Println("MATCH", url, repoPath)
			params := []gin.Param{{"repo", repo}}
			return route.Handler, params
		}
	}

	return nil, nil
}
