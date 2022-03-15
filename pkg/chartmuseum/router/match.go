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
	"net/http"
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
	var noRepoPathSplit []string
	var repo, repoPath, noRepoPath string
	var startIndex, numNoRepoPathParts int
	var tryRepoRoutes bool

	if contextPath != "" {
		if url == contextPath {
			url = "/"
		} else if strings.HasPrefix(url, contextPath) {
			url = strings.Replace(url, contextPath, "", 1)
		} else {
			return nil, nil
		}
	}

	if url == "/health" && method == http.MethodGet {
		for _, route := range routes {
			if route.Path == "/health" {
				return route, nil
			}
		}
	}
	if checkStaticRoute(url) && method == http.MethodGet {
		for _, route := range routes {
			if checkStaticRoute(route.Path) {
				return route, nil
			}
		}
	}

	isApiRoute := checkApiRoute(url)
	if isApiRoute {
		startIndex = 2
	} else {
		startIndex = 1
	}

	pathSplit := strings.Split(url, "/")

	if depthdynamic {
		for _, route := range routes {
			if isApiRoute {
				if !strings.HasPrefix(route.Path, "/api") {
					continue
				}
			} else {
				if strings.HasPrefix(route.Path, "/api") {
					continue
				}
			}
			depth = getDepth(url, route.Path)
			if depth >= 0 {
				break
			}
		}
	}
	if depth < 0 {
		return nil, nil
	}

	if len(pathSplit) >= depth+startIndex {
		repoParts := pathSplit[startIndex : depth+startIndex]
		if len(repoParts) == depth {
			tryRepoRoutes = true
			repo = strings.Join(repoParts, "/")
			noRepoPath = "/" + strings.Join(pathSplit[depth+startIndex:], "/")
			repoPath = "/:repo" + noRepoPath
			if isApiRoute {
				repoPath = "/api" + repoPath
				noRepoPath = "/api" + noRepoPath
			}
			noRepoPathSplit = strings.Split(noRepoPath, "/")
			numNoRepoPathParts = len(noRepoPathSplit)
		}
	}

	for _, route := range routes {
		if route.Method != method {
			continue
		}
		if route.Path == url {
			return route, nil
		} else if tryRepoRoutes {
			if route.Path == repoPath {
				return route, []gin.Param{{"repo", repo}}
			} else {
				p := strings.Replace(route.Path, "/:repo", "", 1)
				if routeSplit := strings.Split(p, "/"); len(routeSplit) == numNoRepoPathParts {
					isMatch := true
					var params []gin.Param
					for i, part := range routeSplit {
						if paramSplit := strings.Split(part, ":"); len(paramSplit) > 1 {
							params = append(params, gin.Param{Key: paramSplit[1], Value: noRepoPathSplit[i]})
						} else if routeSplit[i] != noRepoPathSplit[i] {
							isMatch = false
							break
						}
					}
					if isMatch {
						params = append(params, gin.Param{Key: "repo", Value: repo})
						return route, params
					}
				}
			}
		}
	}

	return nil, nil
}

func checkStaticRoute(url string) bool {
	return strings.HasPrefix(url, "/static")
}

func checkApiRoute(url string) bool {
	return strings.HasPrefix(url, "/api/") && !validRepoRoute.MatchString(url)
}

func splitPath(key string) []string {
	key = strings.Trim(key, "/ ")
	if key == "" {
		return []string{}
	}
	return strings.Split(key, "/")
}

func url2pattern(url string) string {
	urls := splitPath(url)
	var patternItems []string
	for _, item := range urls {
		if strings.HasPrefix(item, ":") {
			if strings.EqualFold(item[1:], "repo") {
				patternItems = append(patternItems, ".*")
			} else {
				patternItems = append(patternItems, `[^/]+`)
			}
		} else {
			patternItems = append(patternItems, item)
		}
	}
	return strings.Replace("^/"+strings.Join(patternItems, "/")+"$",
		"/.*", "(/.*){0,1}", -1)
}

func getDepth(url, routePath string) int {
	r, _ := regexp.Compile(url2pattern(routePath))
	if r.MatchString(url) {
		oriNum := len(strings.Split(routePath, "/"))
		patNum := len(strings.Split(url, "/"))
		return patNum - oriNum + 1
	}
	return -1
}
