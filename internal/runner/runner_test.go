package runner

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
		out, err := exec.Command("podman", "ps", "--all", "--filter", "label=academy.install="+e.Scope, "--format", "{{.ID}}").Output()
		if err != nil || len(strings.TrimSpace(string(out))) != 0 {
			t.Fatal("cancel leaked containers", string(out), err)
		}
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
	t.Run("output-limit", func(t *testing.T) {
		r := run(t, `package main;import "fmt";func main(){for{fmt.Print("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")}}`)
		if r.Status != "output_limit" || len(r.Stdout)+len(r.Stderr) > MaxOutput {
			t.Fatal(r.Status)
		}
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
		if r.Status == "ok" || r.Status == "timeout" {
			t.Fatal("memory was not limited", r)
		}
	})
	t.Run("disk-limit", func(t *testing.T) {
		r := run(t, `package main;import("fmt";"os");func main(){f,e:=os.Create("/work/blob");if e!=nil{panic(e)};b:=make([]byte,1<<20);for i:=0;i<300;i++{if _,e=f.Write(b);e!=nil{fmt.Println("bounded");return}};fmt.Println("unbounded")}`)
		if r.Status != "ok" || r.Stdout != "bounded\n" {
			t.Fatal(r)
		}
	})
	t.Run("pid-limit", func(t *testing.T) {
		r := run(t, `package main;import("fmt";"os/exec");func main(){for i:=0;i<150;i++{if e:=exec.Command("/bin/sleep","2").Start();e!=nil{fmt.Println("bounded");return}};fmt.Println("unbounded")}`)
		if r.Status == "ok" && r.Stdout != "bounded\n" {
			t.Fatal(r)
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
