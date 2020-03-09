# yxorp
A simple reverse proxy for allowing multiple web servers to be accessible via 
a single IP address. It uses the Go 
(`net/http/httputils/ReverseProxy`)[https://golang.org/pkg/net/http/httputil/#ReverseProxy],
so only HTTP/HTTPS works.

## Usage
`yxorp -cfg [path-to-config.json]`

If no `-cfg` flag is provided, a "default" config file is written to the 
directory where the executable exists.

## Note
Due to the working of Go's `ServeMux`, be sure to append a `/` to all domain
names. For example, `example.com/` instead of `example.com`.