package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"academy/internal/projectrunner"
)

func main() {
	socket := flag.String("socket", ".data/project-runner.sock", "private Unix socket")
	goImage := flag.String("go-image", os.Getenv("ACADEMY_P2_GO_IMAGE"), "cached digest-pinned Go image")
	checkerImage := flag.String("checker-image", os.Getenv("ACADEMY_P2_CHECKER_IMAGE"), "cached immutable trusted checker image")
	goMod := flag.String("expected-go-mod", ".data/p2-runtime/go.mod", "operator supplied immutable module lock")
	goSum := flag.String("expected-go-sum", ".data/p2-runtime/go.sum", "operator supplied immutable module checksums")
	flag.Parse()
	if *goImage == "" || *checkerImage == "" {
		log.Fatal("Go and checker immutable images are required")
	}
	if err := ensurePrivateSocketDir(filepath.Dir(*socket)); err != nil {
		log.Fatal(err)
	}
	l, err := projectrunner.Listen(*socket)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { l.Close(); _ = os.Remove(*socket) }()
	readPolicy := func(p string) []byte {
		st, e := os.Stat(p)
		if e != nil {
			log.Fatal("read operator policy: ", e)
		}
		if st.Mode().Perm() != 0600 {
			log.Fatal("operator policy must have mode 0600: ", p)
		}
		b, e := os.ReadFile(p)
		if e != nil {
			log.Fatal("read operator policy: ", e)
		}
		return b
	}
	mod := readPolicy(*goMod)
	sum := readPolicy(*goSum)
	e, err := projectrunner.New(projectrunner.Config{Socket: *socket, GoImage: *goImage, CheckerImage: *checkerImage, ExpectedGoMod: string(mod), ExpectedGoSum: string(sum)})
	if err != nil {
		log.Fatal("project runner unavailable: ", err)
	}
	s := &http.Server{Handler: projectrunner.NewService(e).Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 225 * time.Second, IdleTimeout: 30 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = s.Shutdown(c)
	}()
	if err := s.Serve(l); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func ensurePrivateSocketDir(parent string) error {
	info, err := os.Lstat(parent)
	if os.IsNotExist(err) {
		if err = os.Mkdir(parent, 0700); err != nil {
			return err
		}
		info, err = os.Lstat(parent)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode().Perm() != 0700 {
		return errors.New("project runner socket directory must already be a private 0700 directory")
	}
	return nil
}
