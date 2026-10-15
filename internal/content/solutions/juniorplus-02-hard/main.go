package main

import (
	"net"
	"strconv"
	"strings"
)

func JoinEndpoint(host string, port int) (string, bool) {
	if host == "" || port < 1 || port > 65535 || strings.ContainsAny(host, " /\\\t\r\n") {
		return "", false
	}
	if net.ParseIP(host) == nil {
		if strings.Contains(host, ":") || len(host) > 253 {
			return "", false
		}
		for _, label := range strings.Split(host, ".") {
			if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return "", false
			}
			for _, r := range label {
				if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
					return "", false
				}
			}
		}
	}
	return net.JoinHostPort(host, strconv.Itoa(port)), true
}
