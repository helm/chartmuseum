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
	authorized := false
	responseHeaders := map[string]string{}

	// BasicAuthHeader is only set on the router if ChartMuseum is configured to use
	// basic auth protection. If not set, the server and all its routes are wide open.
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
