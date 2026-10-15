package main

func CountTransitions(values []int) int {
 count:=0; for i:=1;i<len(values);i++ { if values[i]!=values[i-1] { count++ } }; return count
}
