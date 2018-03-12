package multitenant

import (
	"github.com/gin-gonic/gin"
)

var (
	warningHTML = []byte(`<!DOCTYPE html>
<html>
<head>
<title>WARNING</title>
</head>
<body>
<h1>WARNING</h1>
<p>This ChartMuseum install is running in multitenancy mode.</p>
<p>This feature is still and progress, and is not considered stable.</p>
<p>Please run without the --multitenant flag to disable this.</p>
</body>
</html>
	`)
)

func (server *MultiTenantServer) defaultHandler(c *gin.Context) {
	c.Data(200, "text/html", warningHTML)
}
