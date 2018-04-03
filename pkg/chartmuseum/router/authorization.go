package router

import (
	"encoding/base64"
	"net/http"
)

func isRepoAction(act action) bool {
	return act == RepoPullAction || act == RepoPushAction
}

func generateBasicAuthHeader(username string, password string) string {
	base := username + ":" + password
	basicAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(base))
	return basicAuthHeader
}

func (router *Router) authorizeRequest(request *http.Request) (bool, map[string]string) {
	var authorized bool
	responseHeaders := map[string]string{}

	if router.BasicAuthHeader != "" {
		if router.AnonymousGet && request.Method == "GET" {
			authorized = true
		} else if request.Header.Get("Authorization") == router.BasicAuthHeader {
			authorized = true
		} else {
			responseHeaders["WWW-Authenticate"] = "Basic realm=\"ChartMuseum\""
		}
	} else {
		authorized = true
	}

	return authorized, responseHeaders
}
