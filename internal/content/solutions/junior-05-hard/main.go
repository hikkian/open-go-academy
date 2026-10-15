package main
import "errors"
func addMoney(a,b int64)(int64,error){ const max=int64(1<<63-1); const min=-max-1; if b>0&&a>max-b || b<0&&a<min-b{return 0,errors.New("overflow")}; return a+b,nil }
