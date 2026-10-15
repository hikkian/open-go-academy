//go:build ignore

// Produce clearly named, disposable artifact fixtures for visual review only.
package main
import("academy/internal/assessment";"academy/internal/credentials";"academy/internal/credentialrender";"crypto/ed25519";"os";"path/filepath";"strings";"time";"fmt")
func main(){
 _,key,err:=ed25519.GenerateKey(nil);if err!=nil{panic(err)};dir:=".data/m10-certificates";if err=os.MkdirAll(dir,0700);err!=nil{panic(err)}
 for _,exam:=range assessment.All()[1:]{id:=strings.Repeat("a",32);envelope,err:=credentials.Sign(key,credentials.Payload{ID:id,IssuerName:"Open Go Academy — тестовая установка",SubjectName:"Тестовый профиль для визуальной проверки",Rank:exam.Rank,ExamID:exam.ID,CriteriaVersion:exam.Version,CriteriaHash:exam.Hash(),AttemptID:strings.Repeat("b",32),IssuedAt:time.Now().UTC().Format(time.RFC3339Nano),VerifyURL:"http://127.0.0.1:8380/verify/"+id,Competencies:exam.Criteria});if err!=nil{panic(err)}
  pdf,err:=credentialrender.PDF(envelope,false);if err!=nil{panic(err)};png,err:=credentialrender.PNG(envelope,false);if err!=nil{panic(err)}
  for ext,data:=range map[string][]byte{"pdf":pdf,"png":png}{path:=filepath.Join(dir,exam.ID+"."+ext);if err=os.WriteFile(path,data,0600);err!=nil{panic(err)};fmt.Println(path)}
 }
}
