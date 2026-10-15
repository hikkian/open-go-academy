//go:build ignore

// This is an explicit browser-test fixture, never an application endpoint.
// It writes only the disposable .data/e2e database and never executes source.
package main

import (
 "academy/internal/assessment"
 "academy/internal/store"
 "fmt"
 "os"
)
func main(){
 if os.Getenv("ACADEMY_E2E_SEED")!="1" {panic("explicit disposable E2E fixture flag required")}
 s,err:=store.Open(".data/e2e/academy.db");if err!=nil{panic(err)};defer s.Close()
 for _,e:=range assessment.All(){for _,id:=range e.Prerequisites{if _,err=s.RecordAttempt(id,map[string]string{"main.go":"Explicit UI prerequisite fixture; no execution claim"},true,"");err!=nil{panic(err)}}}
 awards,err:=s.RankAwards("local");if err!=nil{panic(err)};ranks:=map[string]bool{};for _,award:=range awards{ranks[award.Rank]=true}
 for _,id:=range []string{"zero-core","junior-core"}{
  e,_:=assessment.Find(id,"");if ranks[e.Rank]{continue}
  if err=s.RegisterExam(e.ID,e.Version,e.Rank,e.Hash());err!=nil{panic(err)}
  required:=[]string{};if e.RequiredRank!=""{required=append(required,e.RequiredRank)}
  attempt,err:=s.StartExam("local",e.ID,e.Version,"ui-fixture-token-hash",e.Prerequisites,required...);if err!=nil{panic(err)}
  if _,err=s.ClaimExamAttempt("local",e.ID,attempt.ID,"ui-fixture-token-hash",0,"{}","{}");err!=nil{panic(err)}
  if _,err=s.FinishExamAttempt("local",attempt.ID,100,true,"[]");err!=nil{panic(err)}
  ranks[e.Rank]=true
 }
 fmt.Println("Seeded disposable browser prerequisites/prior awards only; actual sandbox grading is tested separately")
}
