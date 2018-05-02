This is temporary documentation and it should not be merged

## Configure webhook
* Run Chartmuseum
* Get webhook url
* Create integration on repo `organization/repository/chart-name` by:
```
curl -X POST \
  http://localhost:8080/api/org/repo/integrations \
  -d '{
	"name": "name",
    "url":"...",
    "triggers": [
        "chart:pushed",
        "chart:deleted"
    ],
	"chart": "*"
}'
```
* Push/Delete chart

### Run locally
`dahyphenn/webhook.site` allows you to receive hooks with pretty nice visualization.  
Just run: `docker run -d --init -p 80:80 dahyphenn/webhook.site`

* Validate HMAC using node.js
```
'use strict'

const crypto = require('crypto')
const obj = {} // The paylad your received
const   text = JSON.stringify(obj)
const   key = 'YOUR-SECRET-KEY'
let   hash

hash = crypto.createHmac('sha256', key).update(text).digest('base64')
console.log(hash)
```