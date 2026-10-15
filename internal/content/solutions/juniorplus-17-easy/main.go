package main

import ("net/url";"strings")

func ConfigValid(databaseURL, origin string) bool { db,e:=url.Parse(strings.TrimSpace(databaseURL)); if e!=nil || (db.Scheme!="postgres" && db.Scheme!="postgresql") || db.Hostname()=="" { return false }; u,e:=url.Parse(strings.TrimSpace(origin)); return e==nil && (u.Scheme=="http" || u.Scheme=="https") && u.Hostname()!="" && u.User==nil && (u.Path=="" || u.Path=="/") && u.RawQuery=="" && u.Fragment=="" }
