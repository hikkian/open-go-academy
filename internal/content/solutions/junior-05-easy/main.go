package main
import "fmt"
func formatTotal(cents int64) string { sign:=""; if cents<0 {sign="-"}; mag:=uint64(cents); if cents<0 {mag=uint64(-(cents+1))+1}; return fmt.Sprintf("%s%d.%02d",sign,mag/100,mag%100) }
