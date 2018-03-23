package router

import (
	"fmt"
	"net/http"
	pathutil "path"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"

	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
)

var (
	requestCount         int64
	requestServedMessage = "Request served"
)

func requestWrapper(logger *cm_logger.Logger, engine *gin.Engine, contextPath string, depth int) func(c *gin.Context) {
	return func(c *gin.Context) {
		// The unused c.Request.Response (net/http.Response) is being "abused" due to our need to work around
		// limitations on gin route matching. Please see https://github.com/gin-gonic/gin/issues/388
		// The engine.HandleContext call below is used to "re-match" the incoming route after
		// we have prefixed it with repoRoutePrefix. However, engine.HandleContext in turn calls a c.reset(),
		// which wipes all keys created with c.Set("KeyName"). Instead of relying on c.Set/c.Get, we set
		// key-values in c.Request.Response.Header in the populateContext() method, then afterwards
		// once the augmented request is being handled, we convert them to ordinary context keys
		if c.Request.Response == nil {
			populateContext(c, contextPath, depth)
			engine.HandleContext(c)
			return
		}
		for key, vals := range c.Request.Response.Header {
			c.Set(strings.ToLower(key), vals[0])
		}
		c.Request.Response = nil

		reqPath := c.GetString("originalpath")
		logger.Debugc(c, fmt.Sprintf("Incoming request: %s", reqPath))
		start := time.Now()

		c.Next()

		status := c.Writer.Status()

		meta := []interface{}{
			"path", reqPath,
			"comment", c.Errors.ByType(gin.ErrorTypePrivate).String(),
			"latency", time.Now().Sub(start),
			"clientIP", c.ClientIP(),
			"method", c.Request.Method,
			"statusCode", status,
		}

		switch {
		case status == 200 || status == 201:
			logger.Infoc(c, requestServedMessage, meta...)
		case status == 404:
			logger.Warnc(c, requestServedMessage, meta...)
		default:
			logger.Errorc(c, requestServedMessage, meta...)
		}
	}
}

func populateContext(c *gin.Context, contextPath string, depth int) {
	c.Request.Response = &http.Response{
		Header: http.Header{},
	}

	// set "requestcount" and "requestid" in c.Request.Response.Header
	// add the X-Request-Id header to c.Writer, using one provided by client if present
	reqCount := strconv.FormatInt(atomic.AddInt64(&requestCount, 1), 10)
	c.Request.Response.Header.Set("requestcount", string(reqCount))
	reqID := c.Request.Header.Get("X-Request-Id")
	if reqID == "" {
		reqID = uuid.NewV4().String()
	}
	c.Request.Response.Header.Set("requestid", reqID)
	c.Writer.Header().Set("X-Request-Id", reqID)

	reqPath := c.Request.URL.Path
	c.Request.Response.Header.Set("originalpath", reqPath)

	// if contextPath provided:
	// - if present in request URL, remove it
	// - if missing in request URL, send 404
	if contextPath != "" {
		if strings.HasPrefix(reqPath, contextPath) {
			reqPath = strings.Replace(reqPath, contextPath, "", 1)
		} else {
			c.AbortWithStatus(404)
			return
		}
	}

	// If root route, prefix with rootRoutePrefix
	if reqPath == "/" || reqPath == "/health" {
		c.Request.URL.Path = pathutil.Join(rootRoutePrefix, reqPath)
		return
	}

	pathSplit := strings.Split(reqPath, "/")
	numParts := len(pathSplit)

	if numParts >= 2 && pathSplit[1] == "api" {
		c.Request.URL.Path = pathutil.Join(apiRoutePrefix, reqPath)
	} else {
		// Assume this is a repo route, prefix the route with repoRoutePrefix
		c.Request.URL.Path = pathutil.Join(repoRoutePrefix, reqPath)
	}

	// set "repo" in c.Request.Response.Header (if appropriate)
	if numParts > depth {
		var a []string
		for i := 1; i <= depth; i++ {
			a = append(a, pathSplit[i])
		}
		cmRepoHeader := strings.Join(a, "/")
		c.Request.Response.Header.Set("repo", cmRepoHeader)
	}
}
