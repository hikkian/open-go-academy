package main
func mergeMax(left,right map[string]int) map[string]int { out:=make(map[string]int,len(left)+len(right)); for k,v:=range left { out[k]=v }; for k,v:=range right { if old,ok:=out[k]; !ok || v>old { out[k]=v } }; return out }
