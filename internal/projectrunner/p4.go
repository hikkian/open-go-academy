package projectrunner

import (
	"context"
	"errors"
	"fmt"

	"academy/internal/runner"
)

const p4CheckerVersion = "scalelab-deterministic-v1"

func (e *engine) runP4(ctx context.Context, req Request) (result runner.Result, err error) {
	files := req.Files
	if req.Action == "submit" {
		files = make(map[string]string, len(req.Files)+1)
		for name, source := range req.Files {
			files[name] = source
		}
		files["academy_scalelab_acceptance_test.go"] = p4AcceptanceTests
	}
	archive, err := tarFiles(files)
	if err != nil {
		return runner.Result{}, err
	}
	job, err := randomHex(12)
	if err != nil {
		return runner.Result{}, err
	}
	name := "academy-" + job + "-scalelab"
	args := []string{"run", "--pull=never", "--name", name}
	args = append(args, e.labels(job)...)
	args = append(args, "--stop-timeout=0", "--rm", "--log-driver=k8s-file", "--log-opt=max-size=32kb", "--network", "none", "--user", "65532:65532", "--read-only", "--cap-drop=all", "--security-opt=no-new-privileges", "--http-proxy=false", "--cpus=1", "--memory=1536m", "--memory-swap=1536m", "--pids-limit=128", "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=256m,mode=1777", "--tmpfs", "/work:rw,nosuid,nodev,size=512m,mode=1777", "-e", "HOME=/tmp", "-e", "GOPROXY=off", "-e", "GOSUMDB=off", "-e", "GOTOOLCHAIN=local", "-e", "CGO_ENABLED=0", "-e", "GOMODCACHE=/opt/academy/gomod", "-e", "GOCACHE=/work/cache", "-e", "GOTMPDIR=/work/tmp", "-i", e.cfg.GoImage, "/bin/sh", "-c", p4Command(req.Action))
	cleanup := func() error {
		cc, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()
		return e.removeOwned(cc, "label=academy.project-runner=true", "label=academy.install="+e.install, "label=academy.job="+job)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			e.poisoned.Store(true)
			result = runner.Result{Status: "error", ExitCode: 1}
			err = errors.Join(err, fmt.Errorf("ScaleLab job cleanup failed: %w", cleanupErr))
		}
	}()
	out, runErr := e.run(ctx, archive, args...)
	if runErr != nil {
		if req.Action == "submit" {
			return runner.Result{Status: "error", ExitCode: 1, Stderr: "Детерминированная проверка ScaleLab завершилась ошибкой."}, nil
		}
		return runner.Result{Status: "error", ExitCode: 1, Stderr: bounded(string(out))}, nil
	}
	if req.Action == "submit" {
		return runner.Result{Status: "ok", ExitCode: 0, Passed: true, Checks: []runner.Check{
			{Name: "Baseline и оптимизированная агрегация совпадают на фиксированном наборе", Passed: true},
			{Name: "Пустые входные данные и составные ключи обрабатываются корректно", Passed: true},
			{Name: "Границы uint64 и переполнение проверены без частичного результата", Passed: true},
			{Name: "Go-контракт и детерминированные тесты ScaleLab пройдены", Passed: true},
		}}, nil
	}
	if req.Action == "run" {
		return runner.Result{Status: "ok", ExitCode: 0, Stdout: bounded(string(out)), Passed: false}, nil
	}
	if req.Action == "test" {
		return runner.Result{Status: "ok", ExitCode: 0, Stdout: bounded(string(out)), Passed: false, Checks: []runner.Check{{Name: "Пользовательские Go-тесты выполнены; это не доказательство ранга", Passed: true}}}, nil
	}
	return runner.Result{Status: "error", ExitCode: 1, Stderr: "Неизвестное действие ScaleLab."}, nil
}

func p4Command(action string) string {
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

// Trusted tests execute next to the submitted implementation in the isolated
// Go image. They verify behavior only; timing and benchmark claims stay manual.
const p4AcceptanceTests = `package main
import ("math"; "reflect"; "testing")
func TestAcademyScaleLabContract(t *testing.T) {
 events := []Event{{Service:"api",Region:"west",Bytes:12},{Service:"api",Region:"west",Bytes:0},{Service:"",Region:"",Bytes:8},{Service:"worker",Region:"east",Bytes:5},{Service:"a\x00b",Region:"c"},{Service:"a",Region:"b\x00c"}}
 base, err := AggregateBaseline(events); if err != nil { t.Fatal(err) }
 optimized, err := Aggregate(events); if err != nil { t.Fatal(err) }
 got, err := Aggregate(events); if err != nil { t.Fatal(err) }
 if !reflect.DeepEqual(base, optimized) || !reflect.DeepEqual(base, got) { t.Fatalf("baseline/optimized/API differ: %v / %v / %v",base,optimized,got) }
 if _, ok := base["0:0:"]; !ok { t.Fatal("empty boundary key missing") }
 if len(base) != 5 { t.Fatalf("collision-safe key contract: got %d groups",len(base)) }
 if got, err := Aggregate(nil); err != nil || len(got)!=0 { t.Fatalf("empty input: %v, %v",got,err) }
}
func TestAcademyScaleLabOverflow(t *testing.T) {
 for name, fn := range map[string]func([]Event)(map[string]Totals,error){"baseline":AggregateBaseline,"optimized":Aggregate} {
  t.Run(name,func(t *testing.T){ got,err:=fn([]Event{{Service:"api",Region:"west",Bytes:math.MaxUint64},{Service:"api",Region:"west",Bytes:1}}); if err==nil || got!=nil { t.Fatalf("overflow returned %v, %v",got,err) } })
 }
}
`
