# chartmuseum/auth

[![Codefresh build status]( https://g.codefresh.io/api/badges/pipeline/chartmuseum/chartmuseum%2Fauth%2Fmaster?type=cf-1)]( https://g.codefresh.io/public/accounts/chartmuseum/pipelines/chartmuseum/auth/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/chartmuseum/auth)](https://goreportcard.com/report/github.com/chartmuseum/auth)
[![GoDoc](https://godoc.org/github.com/chartmuseum/auth?status.svg)](https://godoc.org/github.com/chartmuseum/auth)

Go library for generating [ChartMuseum](https://github.com/helm/chartmuseum) JWT Tokens, authorizing HTTP requests, etc.

## How to Use

### Generating a JWT token (example)

[Source](./testcmd/getjwt/main.go)

Clone this repo and run `go run testcmd/getjwt/main.go` to run this example

```go
package main

import (
	"fmt"
	"time"

	cmAuth "github.com/chartmuseum/auth"
)

func main() {

	// This should be the private key associated with the public key used
	// in ChartMuseum server configuration (server.pem)
	cmTokenGenerator, err := cmAuth.NewTokenGenerator(&cmAuth.TokenGeneratorOptions{
		PrivateKeyPath: "./testdata/server.key",
	})
	if err != nil {
		panic(err)
	}

	// Example:
	// Generate a token which allows the user to push to the "org1/repo1"
	// repository, and expires in 5 minutes
	access := []cmAuth.AccessEntry{
		{
			Name:    "org1/repo1",
			Type:    cmAuth.AccessEntryType,
			Actions: []string{cmAuth.PushAction},
		},
	}
	signedString, err := cmTokenGenerator.GenerateToken(access, time.Minute*5)
	if err != nil {
		panic(err)
	}

	// Prints a JWT token which you can use to make requests to ChartMuseum
	fmt.Println(signedString)
}
```

This token will be formatted as a valid JSON Web Token (JWT)
and resemble the following:

```
eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NDM5NTk3OTMsImlhdCI6MTU0Mzk1OTQ5MywiYWNjZXNzIjpbeyJ0eXBlIjoiYXJ0aWZhY3QtcmVwb3NpdG9yeSIsIm5hbWUiOiJvcmcxL3JlcG8xIiwiYWN0aW9ucyI6WyJwdXNoIl19XX0.giLd83d8eK8QbTFnCLmgATV2ohiIb59dIhrg35XYFz-6EHqvirUsfZBdWXMRy2sQUOOIHouVEamv_qErKPbFQYGYureJ9BJmVKA3N2SL8aSiXaa8ZasyjRmayOqri55gNf-LE1XddtO8al6-e6vcXe_0YnkGyfw-ODej83wdoLHjB3VgLGXDdbTyXMJEs0aULmBUxbnyaGFTNWgowfqr8W3Sk64LgRvEJ3gJtTN5r_vjgDDVyMX9SIk0yvlCATN7fJvbiVotoLJTGRKV6PVRN79A16SqSGYsN3Nvym8BUwJgXLPM24ozngje1y2s6YmwOOnKItTIXwU12IqbzlmGRg
```

You can decode this token on [https://jwt.io](http://jwt.io/#id_token=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NDM5NTk3MjYsImlhdCI6MTU0Mzk1OTQyNiwiYWNjZXNzIjpbeyJ0eXBlIjoiYXJ0aWZhY3QtcmVwb3NpdG9yeSIsIm5hbWUiOiJvcmcxL3JlcG8xIiwiYWN0aW9ucyI6WyJwdXNoIl19XX0.mhtlO7RWGIlgZSVecnzJpJ2uKpIfa-6CuC4EADktvdmBq9xWMxauA7yW1wLh0s1cWeRn2wuCj_wXPkFzyjv82A8szEKLkxCGbY8CZqS2xvdDXfOF-NOEqBVfZ_3I0HAE8TGAPNWZ1gSdO4M8gl5ikqsMY60zkt6pzxbKtjXN-RFzq0JI7_2k00uFcFIPg8MId9rWxEE-l-L8t19ieHYnzIT9qJdui1tBXvW6klGIOTScH__mhJ5iko1ExlyjW5qzO84QkI6gogDp0FbKdPs6M6HTXgSCKh22BUtjpgaHNQ-B_wbOvzw3O7ssr_ekVmT3oJ-1p2OIBs7Of-zuMppsOw)
or with something like [jwt-cli](https://github.com/mike-engel/jwt-cli).

The decoded payload of this token will look like the following:
```json
{
  "exp": 1543959726,
  "iat": 1543959426,
  "access": [
    {
      "type": "artifact-repository",
      "name": "org1/repo1",
      "actions": [
        "push"
      ]
    }
  ]
}
```

### Making requests to ChartMuseum

First, obtain the token with the necessary access entries (see example above).

Then use this token to make requests to ChartMuseum,
passing it in the `Authorization` header:

```
> GET /api/charts HTTP/1.1
> Host: localhost:8080
> Authorization: Bearer <token>
```

### Validating a JWT token (example)

[Source](./testcmd/decodejwt/main.go)

Clone this repo and run `go run testcmd/decodejwt/main.go <token>` to run this example

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	cmAuth "github.com/chartmuseum/auth"

	"github.com/dgrijalva/jwt-go"
)

func main() {
	signedString := os.Args[1]

	// This should be the public key associated with the private key used
	// to sign the token
	cmTokenDecoder, err := cmAuth.NewTokenDecoder(&cmAuth.TokenDecoderOptions{
		PublicKeyPath: "./testdata/server.pem",
	})
	if err != nil {
		panic(err)
	}

	token, err := cmTokenDecoder.DecodeToken(signedString)
	if err != nil {
		panic(err)
	}

	// Inspect the token claims as JSON
	c := token.Claims.(jwt.MapClaims)
	byteData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(byteData))
}
```

### Authorizing an incoming request (example)

[Source](./testcmd/authorizer/main.go)

Clone this repo and run `go run testcmd/authorizer/main.go <token>` to run this example

```go
package main

import (
	"fmt"
	"os"

	cmAuth "github.com/chartmuseum/auth"
)

func main() {

	// We are grabbing this from command line, but this should be obtained
	// by inspecting the "Authorization" header of an incoming HTTP request
	signedString := os.Args[1]
	authHeader := fmt.Sprintf("Bearer %s", signedString)

	cmAuthorizer, err := cmAuth.NewAuthorizer(&cmAuth.AuthorizerOptions{
		Realm:         "https://my.site.io/oauth2/token",
		Service:       "my.site.io",
		PublicKeyPath: "./testdata/server.pem",
	})
	if err != nil {
		panic(err)
	}

	// Example:
	// Check if the auth header provided allows access to push to org1/repo1
	permissions, err := cmAuthorizer.Authorize(authHeader, cmAuth.PushAction, "org1/repo1")
	if err != nil {
		panic(err)
	}

	if permissions.Allowed {
		fmt.Println("ACCESS GRANTED")
	} else {

		// If access is not allowed, the WWWAuthenticateHeader will be populated
		// which should be sent back to the client in the "WWW-Authenticate" header
		fmt.Println("ACCESS DENIED")
		fmt.Println(fmt.Sprintf("WWW-Authenticate: %s", permissions.WWWAuthenticateHeader))
	}
}
```

If access denied, the `WWW-Authenticate` header returned will resemble the following:

```
WWW-Authenticate: Bearer realm="https://my.site.io/oauth2/token",service="my.site.io",scope="artifact-repository:org1/repo1:push"
```

## Supported JWT Signing Algorithms

- RS256
