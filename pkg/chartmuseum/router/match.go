package router

import (
	"github.com/gin-gonic/gin"
)

func (router *Router) matchRoute(c *gin.Context) {
	handle, params := match(router.Routes, c.Request.URL.Path)
	if handle == nil {
		c.AbortWithStatus(404)
		return
	}
	c.Params = params
	handle(c)
}

func match(routes []*Route, url string) (gin.HandlerFunc, []gin.Param) {
	for _, route := range routes {
		if route.Path == "/" {
			return route.Handler, nil
		}
	}
	return nil, nil
}
