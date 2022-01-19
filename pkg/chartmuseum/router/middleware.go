/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package router

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

var (
	requestCount         int64
	requestServedMessage = "Request served"
)

func requestWrapper(logger *cm_logger.Logger, logHealth bool, logLatencyInt bool) func(c *gin.Context) {
	return func(c *gin.Context) {
		setupContext(c)

		reqPath := c.Request.URL.EscapedPath()
		logRequest := !strings.HasSuffix(reqPath, "/health") || logHealth
		if logRequest {
			logger.Debugc(c, fmt.Sprintf("Incoming request: %s", reqPath))
		}
		start := time.Now()

		c.Next()

		status := c.Writer.Status()

		meta := []interface{}{
			"path", reqPath,
			"comment", c.Errors.ByType(gin.ErrorTypePrivate).String(),
			"clientIP", c.ClientIP(),
			"method", c.Request.Method,
			"statusCode", status,
		}

		latency := time.Now().Sub(start)
		if logLatencyInt {
			meta = append(meta, []interface{}{"latency", int64(latency)}...)
		} else {
			meta = append(meta, []interface{}{"latency", latency}...)
		}

		switch {
		case status == 200 || status == 201:
			if logRequest {
				logger.Infoc(c, requestServedMessage, meta...)
			}
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
		reqID = uuid.Must(uuid.NewV4()).String()
	}
	c.Set("requestid", reqID)
	c.Writer.Header().Set("X-Request-Id", reqID)
}
