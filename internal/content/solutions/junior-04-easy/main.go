package main
type User struct { ID int; Name string }
func findUser(users []User,id int)(*User,bool){ for i:=range users { if users[i].ID==id{return &users[i],true} }; return nil,false }
