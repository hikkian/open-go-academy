package main
type User struct { ID int; Name string }
func renamedCopy(user *User,name string)(*User,bool){ if user==nil{return nil,false}; copy:=*user; copy.Name=name; return &copy,true }
