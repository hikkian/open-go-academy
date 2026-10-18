package runner

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestValidateFiles(t *testing.T) {
	for _, files := range []map[string]string{nil, {"../main.go": "x"}, {"/main.go": "x"}, {"x_test.go": "x"}, {"main.go": strings.Repeat("x", MaxSource+1)}, {"main.go\x00": "x"}} {
		if ValidateFiles(files) == nil {
			t.Fatalf("accepted invalid files: %v", files)
		}
	}
	if ValidateFiles(map[string]string{"main.go": "package main", "helper.go": "package main"}) != nil {
		t.Fatal("valid multi files rejected")
	}
}
func TestCleanupFailureDisablesNextJob(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	script := "#!/bin/sh\necho \"$1\" >> '" + log + "'\ncase \"$1\" in\nrun) cat >/dev/null; exit 0;;\nrm) exit 1;;\nesac\n"
	if err := os.WriteFile(filepath.Join(dir, "podman"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	e := New()
	q := Request{Mode: "run", Files: map[string]string{"main.go": "package main\nfunc main(){}"}}
	if _, err := e.Execute(context.Background(), q); err == nil {
		t.Fatal("cleanup failure accepted")
	}
	if e.Ready() {
		t.Fatal("health still ready")
	}
	if _, err := e.Execute(context.Background(), q); err == nil {
		t.Fatal("accepted new job after cleanup failure")
	}
	b, _ := os.ReadFile(log)
	if strings.Count(string(b), "run\n") != 1 {
		t.Fatal("second job was launched", string(b))
	}
}
func TestMissingRuntimeFailClosed(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if New().Preflight(context.Background()) == nil {
		t.Fatal("missing runtime admitted")
	}
}
func TestPreflightRejectsMissingCgroupSupport(t *testing.T) {
	for _, tc := range []struct {
		name    string
		info    string
		wantRun bool
	}{
		{name: "cgroup-v1", info: "true v1"},
		{name: "missing-delegation", info: "true v2", wantRun: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			logPath := filepath.Join(dir, "podman-calls")
			t.Setenv("PODMAN_TEST_LOG", logPath)
			script := `#!/bin/sh
printf '%s\n' "$*" >> "$PODMAN_TEST_LOG"
case "$1" in
info) printf '%s\n' 'true v1';;
ps) exit 0;;
run) printf '%s\n' 'missing cgroup delegation' >&2; exit 1;;
rm) exit 0;;
*) exit 42;;
esac
`
			if tc.info == "true v2" {
				script = strings.Replace(script, "true v1", "true v2", 1)
			}
			if err := os.WriteFile(filepath.Join(dir, "podman"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
			if err := New().Preflight(context.Background()); err == nil {
				t.Fatal("preflight admitted unsupported/missing cgroup support")
			}
			b, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}
			calls := strings.Split(strings.TrimSpace(string(b)), "\n")
			if len(calls) == 0 || !strings.HasPrefix(calls[0], "info ") {
				t.Fatalf("unexpected fake runtime calls: %q", b)
			}
			for _, call := range calls {
				if strings.Contains(call, "oga-job-") {
					t.Fatalf("preflight failure started a user job: %q", call)
				}
			}
			if tc.wantRun {
				if len(calls) != 4 || !strings.HasPrefix(calls[1], "ps ") || !strings.HasPrefix(calls[2], "run ") || !strings.Contains(calls[2], "oga-preflight-") || !strings.HasPrefix(calls[3], "rm ") {
					t.Fatalf("missing-delegation probe did not run and clean only preflight: %q", b)
				}
			} else if len(calls) != 1 {
				t.Fatalf("unsupported cgroup v1 must fail before any container starts: %q", b)
			}
			t.Logf("preflight rejected %s; fake runtime calls: %q", tc.name, calls)
		})
	}
}
func TestSandboxIntegration(t *testing.T) {
	if os.Getenv("ACADEMY_SANDBOX_TEST") != "1" {
		t.Skip("set ACADEMY_SANDBOX_TEST=1 with rootless Podman")
	}
	e := New()
	e.Scope = "integration-" + randomID()
	ctx := context.Background()
	if err := e.Preflight(ctx); err != nil {
		t.Fatal(err)
	}
	run := func(t *testing.T, source string) Result {
		t.Helper()
		r, err := e.Execute(ctx, Request{Mode: "run", Files: map[string]string{"main.go": source}})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("status=%s exit=%d stdout=%q stderr=%q", r.Status, r.ExitCode, preview(r.Stdout), preview(r.Stderr))
		return r
	}
	assertNoScopedContainers := func(t *testing.T) {
		t.Helper()
		out, err := exec.Command("podman", "ps", "--all", "--filter", "label=academy.install="+e.Scope, "--format", "{{.ID}}").Output()
		if err != nil || len(strings.TrimSpace(string(out))) != 0 {
			t.Fatalf("scoped runner containers remain: %q (%v)", out, err)
		}
	}
	t.Run("orphan-reconciliation", func(t *testing.T) {
		name := "oga-orphan-" + randomID()
		args := append(e.policy(name), "-d", Image, "sleep", "90")
		if out, err := exec.Command("podman", args...).CombinedOutput(); err != nil {
			t.Fatal(string(out), err)
		}
		if err := e.Preflight(ctx); err != nil {
			t.Fatal(err)
		}
		if exec.Command("podman", "container", "exists", name).Run() == nil {
			t.Fatal("orphan survived restart reconciliation")
		}
	})
	t.Run("cancel-descendants", func(t *testing.T) {
		c, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		start := time.Now()
		r, err := e.Execute(c, Request{Mode: "run", Files: map[string]string{"main.go": "package main;func main(){for{}}"}})
		if err != nil || r.Status != "timeout" || time.Since(start) > 15*time.Second {
			t.Fatal(r, err, time.Since(start))
		}
		assertNoScopedContainers(t)
	})
	t.Run("concurrent-admission-shared-filesystem", func(t *testing.T) {
		firstCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		type execution struct {
			result Result
			err    error
		}
		firstDone := make(chan execution, 1)
		firstSource := `package main
import ("fmt"; "time")
func main() { fmt.Println("first-only"); time.Sleep(6 * time.Second) }`
		go func() {
			r, err := e.Execute(firstCtx, Request{Mode: "run", Files: map[string]string{"main.go": firstSource}})
			firstDone <- execution{result: r, err: err}
		}()
		deadline := time.Now().Add(25 * time.Second)
		for {
			out, err := exec.Command("podman", "ps", "--filter", "label=academy.install="+e.Scope, "--format", "{{.ID}}").Output()
			if err != nil {
				cancel()
				<-firstDone
				t.Fatal("could not observe first scoped job", err)
			}
			if strings.TrimSpace(string(out)) != "" {
				break
			}
			if time.Now().After(deadline) {
				cancel()
				<-firstDone
				t.Fatal("first job did not enter its scoped container before deadline")
			}
			select {
			case completed := <-firstDone:
				t.Fatalf("first job exited before admission check: result=%+v error=%v", completed.result, completed.err)
			case <-time.After(100 * time.Millisecond):
			}
		}
		second, secondErr := e.Execute(ctx, Request{Mode: "run", Files: map[string]string{"main.go": `package main;import "fmt";func main(){fmt.Println("second-only")}`}})
		if secondErr == nil || !strings.Contains(secondErr.Error(), "занят") || second.Stdout != "" || second.Stderr != "" {
			cancel()
			completed := <-firstDone
			t.Fatalf("second job was not rejected cleanly while first held admission: second=%+v error=%v first=%+v/%v", second, secondErr, completed.result, completed.err)
		}
		select {
		case completed := <-firstDone:
			if completed.err != nil || completed.result.Status != "ok" || completed.result.Stdout != "first-only\n" || strings.Contains(completed.result.Stdout, "second-only") {
				t.Fatalf("first job output/status mixed or failed: %+v error=%v", completed.result, completed.err)
			}
		case <-time.After(30 * time.Second):
			cancel()
			<-firstDone
			t.Fatal("first job did not finish within the bounded wait")
		}
		assertNoScopedContainers(t)
		t.Logf("first stdout=%q; second rejected with %v; current-install containers cleaned", "first-only\n", secondErr)
	})
	t.Run("cancel-inherited-pipes", func(t *testing.T) {
		c, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		start := time.Now()
		source := `package main
import "os/exec"
func main() {
	child := exec.Command("/bin/sleep", "60")
	if err := child.Start(); err != nil { panic(err) }
}`
		r, err := e.Execute(c, Request{Mode: "run", Files: map[string]string{"main.go": source}})
		elapsed := time.Since(start)
		if err != nil {
			t.Fatal("runner failed to return after parent exit", err)
		}
		if elapsed > 12*time.Second {
			t.Fatalf("runner blocked on inherited stdout/stderr pipes: %s, result=%+v", elapsed, r)
		}
		assertNoScopedContainers(t)
	})

	t.Run("args-json-and-user-tests", func(t *testing.T) {
		r, err := e.Execute(ctx, Request{Mode: "run", Args: []string{"literal;$(touch injected)", "fixture.json"}, Assets: map[string]string{"fixture.json": "[]"}, Files: map[string]string{"main.go": `package main;import("fmt";"os");func main(){b,e:=os.ReadFile(os.Args[2]);if e!=nil{panic(e)};fmt.Println(os.Args[1]);fmt.Println(string(b))}`}})
		if err != nil || r.Status != "ok" || r.Stdout != "literal;$(touch injected)\n[]\n" {
			t.Fatal(r, err)
		}
		r, err = e.Execute(ctx, Request{Mode: "test", Files: map[string]string{"main.go": "package main;func value()int{return 7}", "main_test.go": `package main;import "testing";func TestValue(t *testing.T){if value()!=7{t.Fatal("bad value")}}`}})
		if err != nil || r.Status != "ok" || r.ExitCode != 0 {
			t.Fatal(r, err)
		}
	})

	t.Run("proxy-env-not-forwarded", func(t *testing.T) {
		t.Setenv("HTTP_PROXY", "http://synthetic-user:synthetic-secret@127.0.0.1:9")
		t.Setenv("HTTPS_PROXY", "http://synthetic-user:synthetic-secret@127.0.0.1:9")
		r := run(t, `package main;import("fmt";"os");func main(){fmt.Println(os.Getenv("HTTP_PROXY")==""&&os.Getenv("HTTPS_PROXY")==""&&os.Getenv("http_proxy")==""&&os.Getenv("https_proxy")=="")}`)
		if r.Status != "ok" || r.Stdout != "true\n" {
			t.Fatal("host proxy environment leaked", r.Status)
		}
	})

	t.Run("multifile", func(t *testing.T) {
		r, err := e.Execute(ctx, Request{Mode: "run", Files: map[string]string{"main.go": `package main;import "fmt";func main(){fmt.Println(sum(2,3))}`, "sum.go": `package main;func sum(a,b int)int{return a+b}`}})
		if err != nil || r.Status != "ok" || r.Stdout != "5\n" {
			t.Fatalf("%+v %v", r, err)
		}
	})
	t.Run("compile-error", func(t *testing.T) {
		r := run(t, `package main;func main(){missing()}`)
		if r.Status != "error" || !strings.Contains(r.Stderr, "undefined") {
			t.Fatal(r)
		}
	})
	t.Run("isolation", func(t *testing.T) {
		sentinel := filepath.Join(t.TempDir(), "host-secret")
		os.WriteFile(sentinel, []byte("ACADEMY_SENTINEL"), 0600)
		r := run(t, `package main;import("fmt";"os";"net";"time";"syscall");func main(){_,a:=os.ReadFile("`+sentinel+`");_,b:=os.Stat("/run/podman/podman.sock");c:=os.WriteFile("/etc/escape",[]byte("x"),0600);conn,d:=net.DialTimeout("tcp","1.1.1.1:443",time.Second);if conn!=nil{conn.Close()};e:=syscall.Mount("none","/work","tmpfs",0,"");fmt.Println(a!=nil,b!=nil,c!=nil,d!=nil,e!=nil)}`)
		if r.Status != "ok" || r.Stdout != "true true true true true\n" {
			t.Fatal(r)
		}
		b, _ := os.ReadFile(sentinel)
		if string(b) != "ACADEMY_SENTINEL" {
			t.Fatal("host sentinel modified")
		}
	})
	t.Run("security-boundary-syscalls", func(t *testing.T) {
		c, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		start := time.Now()
		source := `package main
import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
)
func denied(err error) bool { return errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) }
func main() {
	_, _, unshareErr := syscall.RawSyscall(syscall.SYS_UNSHARE, uintptr(syscall.CLONE_NEWNS), 0, 0)
	mountErr := syscall.Mount("none", "/work", "tmpfs", 0, "size=4096")
	if mountErr == nil { _ = syscall.Unmount("/work", 0) }
	rawFD, _, rawErr := syscall.RawSyscall(syscall.SYS_SOCKET, uintptr(syscall.AF_INET), uintptr(syscall.SOCK_RAW), uintptr(syscall.IPPROTO_ICMP))
	if rawErr == 0 { _ = syscall.Close(int(rawFD)) }
	ptraceErr := syscall.PtraceAttach(1)
	if ptraceErr == nil { _ = syscall.PtraceDetach(1) }
	capEff, seccomp := "", ""
	f, err := os.Open("/proc/self/status")
	if err != nil { panic(err) }
	s := bufio.NewScanner(f)
	for s.Scan() {
		field := strings.Fields(s.Text())
		if len(field) != 2 { continue }
		switch field[0] { case "CapEff:": capEff = field[1]; case "Seccomp:": seccomp = field[1] }
	}
	if err := f.Close(); err != nil { panic(err) }
	if err := s.Err(); err != nil { panic(err) }
	fmt.Printf("unshare=%t mount=%t raw_socket=%t ptrace_pid1=%t capeff=%s seccomp=%s\n", denied(unshareErr), denied(mountErr), denied(rawErr), denied(ptraceErr), capEff, seccomp)
}`
		r, err := e.Execute(c, Request{Mode: "run", Files: map[string]string{"main.go": source}})
		elapsed := time.Since(start)
		assertNoScopedContainers(t)
		if err != nil || r.Status != "ok" || elapsed > 25*time.Second {
			t.Fatalf("bounded security probe failed: elapsed=%s result=%+v error=%v", elapsed, r, err)
		}
		t.Logf("security probe: %s (elapsed=%s)", strings.TrimSpace(r.Stdout), elapsed)
		if !strings.Contains(r.Stdout, "unshare=true mount=true raw_socket=true ptrace_pid1=true") {
			t.Fatalf("expected namespace, mount, raw socket and PID 1 ptrace to be denied: %q", r.Stdout)
		}
		fields := strings.Fields(r.Stdout)
		values := make(map[string]string, len(fields))
		for _, field := range fields {
			key, value, ok := strings.Cut(field, "=")
			if ok {
				values[key] = value
			}
		}
		capValue, parseErr := strconv.ParseUint(values["capeff"], 16, 64)
		if parseErr != nil || capValue != 0 || values["seccomp"] != "2" {
			t.Fatalf("expected zero CapEff and seccomp mode 2, got capeff=%q seccomp=%q (%v)", values["capeff"], values["seccomp"], parseErr)
		}
	})
	t.Run("output-limit", func(t *testing.T) {
		r := run(t, `package main;import "fmt";func main(){for{fmt.Print("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")}}`)
		if r.Status != "output_limit" || len(r.Stdout)+len(r.Stderr) > MaxOutput {
			t.Fatal(r.Status)
		}
	})
	t.Run("output-flood-both-streams", func(t *testing.T) {
		source := `package main
import ("os"; "runtime"; "sync")
func main() {
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, stream := range []*os.File{os.Stdout, os.Stderr} {
		wg.Add(1)
		go func(f *os.File) {
			defer wg.Done()
			<-start
			chunk := make([]byte, 256)
			for { if _, err := f.Write(chunk); err != nil { return }; runtime.Gosched() }
		}(stream)
	}
	close(start)
	wg.Wait()
}`
		r := run(t, source)
		combined := len(r.Stdout) + len(r.Stderr)
		if r.Status != "output_limit" || combined != MaxOutput || len(r.Stdout) == 0 || len(r.Stderr) == 0 {
			t.Fatalf("expected shared output cap across concurrent stdout/stderr floods: stdout=%d stderr=%d status=%s", len(r.Stdout), len(r.Stderr), r.Status)
		}
		assertNoScopedContainers(t)
	})
	t.Run("timeout", func(t *testing.T) {
		start := time.Now()
		r := run(t, `package main;func main(){for{}}`)
		if r.Status != "timeout" || time.Since(start) > e.Timeout+10*time.Second {
			t.Fatal(r.Status, time.Since(start))
		}
	})
	t.Run("memory-limit", func(t *testing.T) {
		r := run(t, `package main;import "time";var blocks [][]byte;func main(){for i:=0;i<100;i++{b:=make([]byte,16<<20);for j:=range b{b[j]=1};blocks=append(blocks,b)};time.Sleep(time.Second)}`)
		if r.Status != "error" || r.ExitCode != 137 {
			t.Fatal("expected cgroup OOM kill exit 137, not a generic failure", r)
		}
	})
	t.Run("disk-limit", func(t *testing.T) {
		r := run(t, `package main;import("fmt";"os");func main(){f,e:=os.Create("/work/blob");if e!=nil{panic(e)};b:=make([]byte,1<<20);for i:=0;i<300;i++{if _,e=f.Write(b);e!=nil{fmt.Println("bounded");return}};fmt.Println("unbounded")}`)
		if r.Status != "ok" || r.Stdout != "bounded\n" {
			t.Fatal(r)
		}
	})
	t.Run("disk-aggregate-many-files", func(t *testing.T) {
		source := `package main
import ("fmt"; "os"; "path/filepath")
func main() {
	chunk := make([]byte, 1<<20)
	created := 0
	for i := 0; i < 384; i++ {
		name := filepath.Join("/work", fmt.Sprintf("aggregate-%03d.bin", i))
		f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
		if err != nil { fmt.Printf("bounded files=%d: %v\n", created, err); return }
		_, writeErr := f.Write(chunk)
		closeErr := f.Close()
		if writeErr != nil || closeErr != nil { fmt.Printf("bounded files=%d: %v/%v\n", created, writeErr, closeErr); return }
		created++
	}
	fmt.Printf("unbounded files=%d\n", created)
}`
		r := run(t, source)
		fields := strings.Fields(r.Stdout)
		created := 0
		if len(fields) > 1 && strings.HasPrefix(fields[1], "files=") {
			created, _ = strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(fields[1], "files="), ":"))
		}
		if r.Status != "ok" || !strings.Contains(r.Stdout, "bounded files=") || strings.Contains(r.Stdout, "unbounded") || created < 100 {
			t.Fatal("expected aggregate tmpfs limit across distinct files", r)
		}
		assertNoScopedContainers(t)
	})
	t.Run("pid-limit", func(t *testing.T) {
		r := run(t, `package main;import("fmt";"os/exec");func main(){var children []*exec.Cmd;for i:=0;i<150;i++{c:=exec.Command("/bin/sleep","2");if e:=c.Start();e!=nil{for _,child:=range children{_=child.Process.Kill()};for _,child:=range children{_=child.Wait()};fmt.Printf("bounded: %v\nchildren-cleaned:%d\n",e,len(children));return};children=append(children,c)};for _,child:=range children{_=child.Process.Kill()};for _,child:=range children{_=child.Wait()};fmt.Println("unbounded")}`)
		if r.Status != "ok" || r.ExitCode != 0 || !strings.Contains(r.Stdout, "resource temporarily unavailable") ||
			!strings.Contains(r.Stdout, "children-cleaned:") || strings.Contains(r.Stdout, "unbounded") {
			t.Fatal("expected PID cgroup EAGAIN after reaping children", r)
		}
	})
	t.Run("format", func(t *testing.T) {
		r, err := e.Execute(ctx, Request{Mode: "format", Files: map[string]string{"main.go": "package main;func main(){}"}})
		if err != nil || r.Status != "ok" || !strings.Contains(r.Files["main.go"], "func main() {}") {
			t.Fatal(r, err)
		}
	})
}

func preview(s string) string {
	if len(s) > 256 {
		return s[:256] + "… (truncated)"
	}
	return s
}

func TestConcurrencyAdmissionIsBounded(t *testing.T) {
	e := New()
	e.slots <- struct{}{}
	if _, err := e.Execute(context.Background(), Request{Mode: "run", Files: map[string]string{"main.go": "package main;func main(){}"}}); err == nil {
		t.Fatal("overload was admitted")
	}
	<-e.slots
}

func TestPreflightCleanupFailureReturnsError(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
case "$1" in
info) echo 'true v2';;
ps) exit 0;;
run) exit 0;;
rm) exit 1;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "podman"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	e := New()
	if e.Preflight(context.Background()) == nil || e.Ready() {
		t.Fatal("failed preflight cleanup admitted runner")
	}
}
func TestClientPreservesLargeEscapedFormatResult(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "runner.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	source := "package main\n\nfunc main() {}\n/*" + strings.Repeat("<>&", 20000) + "*/\n"
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req) != nil {
			http.Error(w, "invalid", 400)
			return
		}
		json.NewEncoder(w).Encode(Result{Status: "ok", Files: req.Files})
	})}
	go server.Serve(listener)
	defer server.Close()
	result, err := NewClient(socket).Execute(context.Background(), Request{Mode: "format", Files: map[string]string{"main.go": source}})
	if err != nil || result.Files["main.go"] != source {
		t.Fatal("valid64KiB source lost by JSON escaping/response cap", err)
	}
}
