package main

import (
	"net"
	"strconv"
)

type Endpoint struct {
	Host string
	Port int
}

func ParseEndpoint(address string) (Endpoint, bool) {
	h, p, e := net.SplitHostPort(address)
	if e != nil || h == "" || p == "" {
		return Endpoint{}, false
	}
	for _, r := range p {
		if r < '0' || r > '9' {
			return Endpoint{}, false
		}
	}
	n, e := strconv.Atoi(p)
	if e != nil || n < 1 || n > 65535 {
		return Endpoint{}, false
	}
	return Endpoint{Host: h, Port: n}, true
}
