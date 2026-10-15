package main
import "errors"
func discounted(price,percent int64)(int64,error){ if price<0||percent<0||percent>100{return 0,errors.New("invalid")}; return price-price/100*percent-(price%100)*percent/100,nil }
