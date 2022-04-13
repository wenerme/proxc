# proxc

Proxy Server with Cache in Golang

```bash
go install github.com/wenerme/proxc/cmd/proxc@latest
proxc config                # current config
DB_DIR=/tmp/proxc/db proxc

curl -x 127.0.0.1:9080 https://wener.me -v > /dev/null
curl -x 127.0.0.1:9080 https://wener.me -v > /dev/null  # X-From-Cache: 1

# cached data
sqlite3 /tmp/proxc/db/wener.me.sqlite 'select method,url,raw_size,body_size,length(body) from http_responses'
```

- Default to SQLite Backend - One SQLite DB per Host + File DB
- Default to zstd compressed - `--encoding=zstd`
- httpcache based on https://github.com/gregjones/httpcache
- proxy based on https://github.com/lqqyt2423/go-mitmproxy

## Support Encoding

```bash
# initial
curl -s https://wener.me | sha256sum
curl -sx 127.0.0.1:9080 https://wener.me -vk | sha256sum
# cached - X-From-Cache: 1
curl -sx 127.0.0.1:9080 https://wener.me -vk | sha256sum
curl -sx 127.0.0.1:9080 https://wener.me -vk --compressed | sha256sum

curl --version | grep brotli
curl -sx 127.0.0.1:9080 https://wener.me -vk --compressed -H 'Accept-Encoding: br' | sha256sum
```
