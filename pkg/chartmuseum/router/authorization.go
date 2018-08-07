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
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
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
	} else if router.BearerAuthHeader != "" {
		if router.AnonymousGet && request.Method == "GET" {
			authorized = true
		} else {
			if request.Header.Get("Authorization") != "" {
				splitToken := strings.Split(request.Header.Get("Authorization"), "Bearer ")
				_, isValid := validateJWT(splitToken[1], router)
				if isValid {
					authorized = true
				}
			} else {
				// TODO: needs work: Should I redirect to Auth Server? or just error as it does now.
				// FIXME: I should probably move the scope out of this query string parsing as it is parsing * unnecessarly.
				queryString := url.PathEscape("service=" + router.AuthService + "&scope=registry:catalog:" + router.AuthScopes)
				responseHeaders["WWW-Authenticate"] = "Bearer realm=\"" + router.AuthRealm + "?" + queryString + "\""
			}
		}
	} else {
		authorized = true
	}
	
	return authorized, responseHeaders
}

// verify if JWT is valid by using the rsa public certificate pem
// currently this only works with RSA key signing
// TODO: how best to handle many different signing algorithms?
func validateJWT(t string, router *Router) (*jwt.Token, bool) {
	valid := false

	key, err := getRSAKey(router.AuthPublicCert)
	if err != nil {
		fmt.Println(err)
	}

	token, err := jwt.Parse(t, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})
	if err != nil {
		fmt.Println("Token parse error: ", err)
	} else {
		fmt.Println("token is valid")
		valid = true
	}
	return token, valid
}

// https://github.com/dgrijalva/jwt-go/blob/master/rsa_test.go
func getRSAKey(key []byte) (*rsa.PublicKey, error) {
	parsedKey, err := jwt.ParseRSAPublicKeyFromPEM(key)
	if err != nil {
		fmt.Println("error parsing RSA key from PEM: ", err)
	}

	return parsedKey, nil
}

// Load authorization server public pem file
// TODO: have this be fetched from a url instead of file
func loadPublicCertFromFile(certPath string, router *Router) {
	publicKey, err := ioutil.ReadFile(certPath)
	if err != nil {
		router.Logger.Fatal("Error reading Public Key")
	}
	router.AuthPublicCert = publicKey
}
