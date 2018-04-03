package router

import (
	"fmt"
	"strconv"
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

func requestWrapper(logger *cm_logger.Logger) func(c *gin.Context) {
	return func(c *gin.Context) {
		setupContext(c)

		reqPath := c.Request.URL.Path
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

func setupContext(c *gin.Context) {
	reqCount := strconv.FormatInt(atomic.AddInt64(&requestCount, 1), 10)
	c.Set("requestcount", reqCount)
	reqID := c.Request.Header.Get("X-Request-Id")
	if reqID == "" {
		reqID = uuid.NewV4().String()
	}
	c.Set("requestid", reqID)
	c.Writer.Header().Set("X-Request-Id", reqID)
}
