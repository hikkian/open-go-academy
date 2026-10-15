package main

import ("net";"strings")

func IsLoopbackBind(host string) bool { if strings.EqualFold(host,"localhost") { return true }; ip:=net.ParseIP(host); return ip!=nil && ip.IsLoopback() }
