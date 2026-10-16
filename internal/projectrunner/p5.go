package projectrunner

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"academy/internal/runner"
)

const p5CheckerVersion = "capstone-deterministic-v1"
const p5ArtifactPolicy = "capstone-required-docs-v1:requirements,architecture,threat,operations,postmortem,and-three-adr-files;nonempty-only"

var p5ADRName = regexp.MustCompile(`^design/adr/ADR-[0-9]{3}-.+\.md$`)

func (e *engine) runP5(ctx context.Context, req Request) (result runner.Result, err error) {
	files := req.Files
	if req.Action == "submit" {
		files = make(map[string]string, len(req.Files)+1)
		for name, source := range req.Files {
			files[name] = source
		}
		files["academy_capstone_acceptance_test.go"] = p5AcceptanceTests
	}
	archive, err := tarFiles(files)
	if err != nil {
		return runner.Result{}, err
	}
	job, err := randomHex(12)
	if err != nil {
		return runner.Result{}, err
	}
	name := "academy-" + job + "-capstone"
	args := []string{"run", "--pull=never", "--name", name}
	args = append(args, e.labels(job)...)
	args = append(args, "--stop-timeout=0", "--rm", "--log-driver=k8s-file", "--log-opt=max-size=32kb", "--network", "none", "--user", "65532:65532", "--read-only", "--cap-drop=all", "--security-opt=no-new-privileges", "--http-proxy=false", "--cpus=1", "--memory=1536m", "--memory-swap=1536m", "--pids-limit=128", "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=256m,mode=1777", "--tmpfs", "/work:rw,nosuid,nodev,size=512m,mode=1777", "-e", "HOME=/tmp", "-e", "GOPROXY=off", "-e", "GOSUMDB=off", "-e", "GOTOOLCHAIN=local", "-e", "CGO_ENABLED=0", "-e", "GOMODCACHE=/opt/academy/gomod", "-e", "GOCACHE=/work/cache", "-e", "GOTMPDIR=/work/tmp", "-i", e.cfg.GoImage, "/bin/sh", "-c", p5Command(req.Action))
	cleanup := func() error {
		cc, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()
		return e.removeOwned(cc, "label=academy.project-runner=true", "label=academy.install="+e.install, "label=academy.job="+job)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			e.poisoned.Store(true)
			result = runner.Result{Status: "error", ExitCode: 1}
			err = errors.Join(err, fmt.Errorf("Capstone job cleanup failed: %w", cleanupErr))
		}
	}()
	out, runErr := e.run(ctx, archive, args...)
	if runErr != nil {
		if req.Action == "submit" {
			return runner.Result{Status: "error", ExitCode: 1, Stderr: "Детерминированная проверка Capstone завершилась ошибкой."}, nil
		}
		return runner.Result{Status: "error", ExitCode: 1, Stderr: bounded(string(out))}, nil
	}
	if req.Action == "run" {
		return runner.Result{Status: "ok", ExitCode: 0, Stdout: bounded(string(out)), Passed: false}, nil
	}
	if req.Action == "test" {
		return runner.Result{Status: "ok", ExitCode: 0, Stdout: bounded(string(out)), Passed: false, Checks: []runner.Check{{Name: "Пользовательские Go-тесты прототипа выполнены; это не подтверждение архитектуры, human review или ранга", Passed: true}}}, nil
	}
	checks := p5ArtifactChecks(req.Files)
	checks = append(checks,
		runner.Check{Name: "Детерминированные проверки идемпотентности и tenant-scope прототипа пройдены (machine evidence)", Passed: true},
		runner.Check{Name: "Конкурентная выдача не дублирует действующую lease (machine evidence)", Passed: true},
		runner.Check{Name: "Истечение lease и stale fencing-token отклонены (machine evidence)", Passed: true},
		runner.Check{Name: "Эта оценка касается только machine evidence; independent authenticated human review и rank gate не проверялись и остаются обязательными", Passed: true},
	)
	machinePassed := true
	for _, check := range checks {
		machinePassed = machinePassed && check.Passed
	}
	return runner.Result{Status: "ok", ExitCode: 0, Passed: machinePassed, Checks: checks}, nil
}

func p5Command(action string) string {
	cmd := "mkdir -p /work/src /work/cache /work/tmp && cd /work/src && tar -xf -"
	switch action {
	case "run":
		return cmd + " && go run -mod=readonly ."
	case "test":
		return cmd + " && go test -mod=readonly ./..."
	default:
		return cmd + " && go test -mod=readonly -count=1 ./..."
	}
}

func p5ArtifactChecks(files map[string]string) []runner.Check {
	required := []string{"docs/project.md", "docs/rubric.md", "docs/threat-model.md", "design/requirements.md", "design/architecture-template.md", "design/operations-template.md", "design/postmortem-template.md", "design/threat-template.md"}
	checks := make([]runner.Check, 0, 2)
	artifactsPresent := true
	for _, name := range required {
		if strings.TrimSpace(files[name]) == "" {
			artifactsPresent = false
		}
	}
	checks = append(checks, runner.Check{Name: "Обязательные архитектурные документы присутствуют и непусты (структурная machine check)", Passed: artifactsPresent})
	adrCount := 0
	for name, body := range files {
		if p5ADRName.MatchString(name) && strings.TrimSpace(body) != "" {
			adrCount++
		}
	}
	checks = append(checks, runner.Check{Name: "Присутствуют как минимум три отдельные ADR-записи (структурная machine check)", Passed: adrCount >= 3})
	return checks
}

// The versioned test overlay is stored in the profile hash. It tests the
// deterministic in-memory risk model only; it cannot establish datastore,
// external-effect, deployment, security, or SLO guarantees.
const p5AcceptanceTests = `package main
import ("sync"; "testing"; "time")
func TestAcademyCapstoneEnqueueTenantIdempotency(t *testing.T) {
 q:=NewQueue(); event:=Event{ID:"evt-7",TenantID:"tenant-a",Payload:[]byte("v1")}
 inserted,err:=q.Enqueue(event); if err!=nil||!inserted { t.Fatalf("first enqueue=%v,%v",inserted,err) }
 inserted,err=q.Enqueue(event); if err!=nil||inserted { t.Fatalf("replay=%v,%v",inserted,err) }
 conflict:=event; conflict.Payload=[]byte("v2"); if _,err=q.Enqueue(conflict); err==nil { t.Fatal("different payload replay was accepted") }
 other:=event; other.TenantID="tenant-b"; if inserted,err=q.Enqueue(other); err!=nil||!inserted { t.Fatalf("other tenant=%v,%v",inserted,err) }
}
func TestAcademyCapstoneConcurrentClaim(t *testing.T) {
 q:=NewQueue(); if _,err:=q.Enqueue(Event{ID:"evt",TenantID:"t",Payload:[]byte("p")});err!=nil { t.Fatal(err) }
 now:=time.Unix(1800000000,0); const n=24; var wg sync.WaitGroup; var mu sync.Mutex; claims:=0
 for i:=0;i<n;i++ { wg.Add(1); go func(i int){ defer wg.Done(); c,err:=q.Claim(string(rune('A'+i)),now,time.Minute); if err!=nil { t.Errorf("Claim: %v",err); return }; if c!=nil { mu.Lock(); claims++; mu.Unlock() } }(i) }; wg.Wait()
 if claims!=1 { t.Fatalf("simultaneous live claims=%d",claims) }
}
func TestAcademyCapstoneLeaseExpiryAndFencing(t *testing.T) {
 q:=NewQueue(); event:=Event{ID:"evt",TenantID:"t",Payload:[]byte("p")}; if _,err:=q.Enqueue(event);err!=nil { t.Fatal(err) }
 start:=time.Unix(1800000000,0); old,err:=q.Claim("old",start,10*time.Second); if err!=nil||old==nil { t.Fatalf("old claim=%v,%v",old,err) }
 if err=q.Complete(event.ID,event.TenantID,old.FenceToken,start.Add(10*time.Second)); err==nil { t.Fatal("completion at lease boundary accepted") }
 current,err:=q.Claim("new",start.Add(10*time.Second),time.Minute); if err!=nil||current==nil { t.Fatalf("reclaim=%v,%v",current,err) }
 if current.FenceToken<=old.FenceToken { t.Fatalf("fence did not increase: %d <= %d",current.FenceToken,old.FenceToken) }
 if err=q.Complete(event.ID,event.TenantID,old.FenceToken,start.Add(11*time.Second)); err==nil { t.Fatal("stale token accepted") }
 if err=q.Complete(event.ID,event.TenantID,current.FenceToken,start.Add(11*time.Second)); err!=nil { t.Fatalf("valid completion: %v",err) }
 if state,ok:=q.State(event.TenantID,event.ID); !ok||state!="done" { t.Fatalf("final state=%q,%v",state,ok) }
}
func TestAcademyCapstoneValidation(t *testing.T) {
 q:=NewQueue(); if _,err:=q.Enqueue(Event{}); err==nil { t.Fatal("empty event accepted") }
 if _,err:=q.Claim("",time.Unix(1,0),time.Second); err==nil { t.Fatal("empty worker accepted") }
 if _,err:=q.Claim("worker",time.Unix(1,0),0); err==nil { t.Fatal("zero lease accepted") }
 if err:=q.Complete("missing","t",1,time.Unix(2,0)); err==nil { t.Fatal("unclaimed event completed") }
}
`
