package main

func NextNetworkStage(d, t, l, h bool) string {
	if !d {
		return "dns"
	}
	if !t {
		return "tcp"
	}
	if !l {
		return "tls"
	}
	if !h {
		return "http"
	}
	return "complete"
}
