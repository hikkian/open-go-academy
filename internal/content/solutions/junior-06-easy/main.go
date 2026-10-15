package main
import "strconv"
func parsePositiveID(text string)(int,error){ id,err:=strconv.Atoi(text); if err!=nil||id<=0{return 0,strconv.ErrSyntax}; return id,nil }
