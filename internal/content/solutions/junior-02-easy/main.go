package main
func lookup(scores map[string]int, name string) (int,bool) { v,ok:=scores[name]; return v,ok }
