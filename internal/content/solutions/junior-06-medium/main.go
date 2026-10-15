package main
import ("fmt";"strconv";"strings")
type Record struct { ID int; Name string }
func parseRecord(line string)(Record,error){ i:=strings.IndexByte(line,','); if i<0||strings.Contains(line[i+1:],","){return Record{},fmt.Errorf("record: %w",strconv.ErrSyntax)}; id,e:=strconv.Atoi(line[:i]); if e!=nil{return Record{},fmt.Errorf("id: %w",e)}; name:=strings.TrimSpace(line[i+1:]); if id<=0||name==""{return Record{},fmt.Errorf("record: %w",strconv.ErrSyntax)}; return Record{ID:id,Name:name},nil }
