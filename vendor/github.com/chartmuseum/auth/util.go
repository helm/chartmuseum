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

package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"io/ioutil"
)

func containsAction(actionsList []string, action string) bool {
	for _, a := range actionsList {
		if a == action {
			return true
		}
	}
	return false
}

func generateBasicAuthHeader(username string, password string) string {
	base := username + ":" + password
	basicAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(base))
	return basicAuthHeader
}

func generatePublicKey(publicKeyPath string, publicKey []byte) (*rsa.PublicKey, error) {
	pem, err := getPemFromPathOrKey(publicKeyPath, publicKey)
	if err != nil {
		return nil, err
	}

	// https://github.com/dgrijalva/jwt-go/blob/master/rsa_test.go
	return jwt.ParseRSAPublicKeyFromPEM(pem)
}

func generatePrivateKey(privateKeyPath string, privateKey []byte) (*rsa.PrivateKey, error) {
	pem, err := getPemFromPathOrKey(privateKeyPath, privateKey)
	if err != nil {
		return nil, err
	}

	return jwt.ParseRSAPrivateKeyFromPEM(pem)
}

func getPemFromPathOrKey(keyPath string, key []byte) ([]byte, error) {
	var pem []byte
	var err error

	if keyPath != "" {
		pem, err = ioutil.ReadFile(keyPath)
		if err != nil {
			return nil, err
		}
	} else if key != nil {
		pem = key
	} else {
		return nil, errors.New("must supply either cert path or cert")
	}

	return pem, nil
}

func getTokenCustomClaims(token *jwt.Token) (*Claims, error) {
	c := token.Claims.(jwt.MapClaims)
	byteData, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	claims := Claims{}
	json.Unmarshal(byteData, &claims)
	return &claims, nil
}
