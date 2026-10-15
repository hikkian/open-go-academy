package main
func unique(values []int) []int { out:=make([]int,0,len(values)); seen:=make(map[int]struct{},len(values)); for _,v:=range values { if _,ok:=seen[v]; !ok { seen[v]=struct{}{}; out=append(out,v) } }; return out }
