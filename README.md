# proxc

Proxy Server with Cache

```bash
go install github.com/wenerme/proxc/cmd/proxc@latest
proxc config                # current config
DB_DIR=/tmp/proxc/db proxc

curl -x 127.0.0.1:9080 https://wener.me -v > /dev/null
curl -x 127.0.0.1:9080 https://wener.me -v > /dev/null  # X-From-Cache: 1

# cached data
sqlite3 /tmp/proxc/db/wener.me.sqlite 'select method,url,raw_size,body_size,length(body) from http_responses'
```

- Default to SQLite Backend
  - One SQLite DB per Host + File DB
- httpcache based on https://github.com/gregjones/httpcache
- proxy based on https://github.com/lqqyt2423/go-mitmproxy
