package main
func keepAtLeast(values []int, minimum int) []int { out:=make([]int,0,len(values)); for _,v:=range values { if v>=minimum { out=append(out,v) } }; return out }
