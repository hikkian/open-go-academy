package main

import "net/url"

func IsHTTPSOrigin(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Scheme == "https" && u.Host != ""
}
