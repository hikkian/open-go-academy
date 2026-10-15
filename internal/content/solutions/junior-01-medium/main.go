package main
import "errors"
func sum(values []int) (int, error) { total:=0; max:=int(^uint(0)>>1); min:=-max-1; for _, v := range values { if v > 0 && total > max-v || v < 0 && total < min-v { return 0, errors.New("overflow") }; total += v }; return total,nil }
