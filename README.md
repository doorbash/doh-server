## Build
```
GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -trimpath -ldflags="-s -w" -o doh-server
```

## Usage
```
./doh-server [OPTIONS]
```

**Options:**
```
  -addr string
        Listen address (default "localhost:53")
  -debug
        print debug logs
  -dohserver string
        DNS Over HTTPS server address (default "mozilla.cloudflare-dns.com")
  -proxy string
        Http proxy
  -timeout duration
        timeout (default 10s) (default 10s)
```

## Example
```
doh-server -addr 0.0.0.0:53 -dohserver 1.1.1.1 -proxy http://127.0.0.1:10809 -timeout 10s
```
