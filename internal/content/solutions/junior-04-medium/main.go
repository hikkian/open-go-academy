package main
type User struct { ID int; Active bool }
func activeUsers(users []User)map[int]bool{ out:=make(map[int]bool); for _,u:=range users{out[u.ID]=u.Active}; return out }
