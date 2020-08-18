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
	"regexp"
	"strings"
	"github.com/gin-gonic/gin"
)

var (
	validRepoRoute = regexp.MustCompile(`^.*\.(yaml|tgz|prov)$`)
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
func match(routes []*Route, method string, url string, contextPath string, depth int, depthdynamic bool) (*Route, []gin.Param) {
	// strip off contextPath prefix
	if contextPath != "" {
		if url == contextPath {
			url = "/"
		} else if strings.HasPrefix(url, contextPath) {
			url = strings.Replace(url, contextPath, "", 1)
		} else {
			return nil, nil
		}
	}

	isApiRequest := checkApiRoute(url)

	requestPathComponents := splitPath(url)
	var routePathComponents []string
	var route *Route
	for _, routeCandidate := range routes {
		// if the methods don't match up, skip this route
		if method != routeCandidate.Method {
			continue
		}
		routePathComponents = splitPath(routeCandidate.Path)
		// if the request is an API request and the route is not an API route, skip this route
		if isApiRequest && (len(routePathComponents) > 0 && routePathComponents[0] != "api") {
			continue
		}
		// if the request is not an API request and the route is an API route, skip it
		if !isApiRequest && (len(routePathComponents) > 0 && routePathComponents[0] == "api") {
			continue
		}

		if match, pathComponents, depthCandidate := comparePaths(
			requestPathComponents,
			routePathComponents,
			":repofragment",
		); match {
			if depthdynamic || depthCandidate < depth {
				route = routeCandidate
				requestPathComponents = pathComponents
			}
			break
		}

		if match, pathComponents, depthCandidate := comparePaths(
			requestPathComponents,
			routePathComponents,
			":repo",
		); match {
			if depthCandidate == -1 || depthdynamic || depthCandidate == depth {
				route = routeCandidate
				requestPathComponents = pathComponents
			}
			break
		}
	}
	if route != nil {
		params := []gin.Param{}
		for i, _ := range routePathComponents {
			if strings.HasPrefix(routePathComponents[i], ":") {
				params = append(params, gin.Param{
					Key: routePathComponents[i][1:],
					Value: requestPathComponents[i],
				})
			}
		}
		if len(params) == 0 {
			params = nil
		}
		return route, params
	}
	return nil, nil
}

/*
Given a list of request and route path components and a "variable length
placeholder" name, meaning a placeholder that can represent multiple path
components; determine whether the request matches the route.

Returns a boolean, as well as a modified requestPathComponents, with the
multiple values represented by the variable length placholder combined into a
single element delimited by slashes, and an int representing the number of
components that were combined.
*/
func comparePaths(requestPathComponents []string, routePathComponents []string, variableLengthPlaceholder string) (bool, []string, int) {
	if len(routePathComponents) == 0 {
		if len(requestPathComponents) == 0 {
			return true, requestPathComponents, -1
		} else {
			return false, []string{}, -1
		}
	}
	var placeholderLength int
	if startIndex := indexOfString(routePathComponents, variableLengthPlaceholder); startIndex >= 0 {
		placeholderLength = len(requestPathComponents) - (len(routePathComponents) - 1)
		if placeholderLength < 0 {
			return false, []string{}, -1
		}
		// replace the slice of request path component strings representing the
		// variable length placeholder with a single slash-delimited string
		var newRequestPathComponents []string
		for _, component := range requestPathComponents[:startIndex] {
			newRequestPathComponents = append(newRequestPathComponents, component)
		}
		newRequestPathComponents = append(newRequestPathComponents, strings.Join(requestPathComponents[startIndex:startIndex + placeholderLength], "/"))
		for _, component := range requestPathComponents[startIndex + placeholderLength:] {
			newRequestPathComponents = append(newRequestPathComponents, component)
		}
		requestPathComponents = newRequestPathComponents
	} else {
		// if the route we're looking at doesn't have a variable length
		// placeholder, it's only valid if the request and route paths have the
		// same number of components
		if len(requestPathComponents) != len(routePathComponents) {
			return false, []string{}, -1
		}
		placeholderLength = -1
	}
	for i, _ := range routePathComponents {
		if strings.HasPrefix(routePathComponents[i], ":") {
			continue
		}
		if routePathComponents[i] != requestPathComponents[i] {
			return false, []string{}, -1
		}
	}
	return true, requestPathComponents, placeholderLength
}

func splitPath(key string) []string {
	key = strings.Trim(key, "/")
	if key == "" {
		return []string{}
	}
	return strings.Split(key, "/")
}

func indexOfString(slice []string, str string) int {
	for index, item := range(slice) {
		if item == str {
			return index
		}
	}
	return -1
}

func checkApiRoute(url string) bool {
	return strings.HasPrefix(url, "/api/") && !validRepoRoute.MatchString(url)
}
