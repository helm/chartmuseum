package router

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"

	"github.com/gin-gonic/gin"
)

func loggingMiddleware(logger *cm_logger.Logger) gin.HandlerFunc {
	var requestCount int64
	return func(c *gin.Context) {
		reqCount := strconv.FormatInt(atomic.AddInt64(&requestCount, 1), 10)
		c.Set("RequestCount", reqCount)

		reqPath := c.Request.URL.Path
		logger.Debugc(c, fmt.Sprintf("Incoming request: %s", reqPath))
		start := time.Now()
		c.Next()

		msg := "Request served"
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
			logger.Infoc(c, msg, meta...)
		case status == 404:
			logger.Warnc(c, msg, meta...)
		default:
			logger.Errorc(c, msg, meta...)
		}
	}
}
