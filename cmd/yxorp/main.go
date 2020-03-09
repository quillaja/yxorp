package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/quillaja/yxorp/log"
)

func main() {
	const defaultCfgFile = "default.json"
	cfgFile := flag.String("cfg", defaultCfgFile, "path to config file")
	flag.Parse()

	if *cfgFile == defaultCfgFile {
		fmt.Println("no config file specified")
		err := writeConfig(config{
			Listen: []l{{"", "80"}},
			Paths:  []p{{"example.com/", "http://localhost:8000"}},
		}, *cfgFile)
		if err != nil {
			fmt.Printf("error when writing default config: %s\n", err)
		}
		os.Exit(1)
	}

	cfg, err := readConfig(*cfgFile)
	if err != nil {
		log.Printf(log.Critical, "couldn't read config file %s: %s\n", *cfgFile, err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	for _, path := range cfg.Paths {
		u, err := url.Parse(path.Internal)
		if err != nil {
			log.Print(log.Error, "couldn't parse %s: %s", path.Internal, err)
			continue
		}
		revprox := httputil.NewSingleHostReverseProxy(u)
		revprox.ErrorLog = stdlog.New(os.Stdout, fmt.Sprintf("<%d>", log.Error), stdlog.Lshortfile)
		mux.HandleFunc(path.External, func(w http.ResponseWriter, req *http.Request) {
			log.Printf(log.Info, "%s -> %s", req.URL.String(), u.String())
			revprox.ServeHTTP(w, req)
		})
		log.Printf(log.Info, "sending %s -> %s\n", path.External, path.Internal)
	}

	servers := make([]*http.Server, 0, len(cfg.Listen))
	var wg sync.WaitGroup
	for _, addr := range cfg.Listen {
		s := http.Server{
			Addr:    addr.Host + ":" + addr.Port,
			Handler: mux,
		}
		var err error
		go func() {
			err = s.ListenAndServe()
		}()
		if err != nil {
			log.Printf(log.Error, "failed to listen on %s:%s: %s\n", addr.Host, addr.Port, err)
			s.Close()
			continue
		}
		servers = append(servers, &s)
		wg.Add(1)
		log.Printf(log.Info, "listening on %s:%s\n", addr.Host, addr.Port)
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-sig

	shutdownWait, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, s := range servers {
		go func(s *http.Server) {
			s.Shutdown(shutdownWait)
			wg.Done()
			log.Print(log.Info, "shutdown server for", s.Addr)
		}(s)
	}
	wg.Wait()
}

type config struct {
	Listen []l
	Paths  []p
}

type l struct {
	Host string
	Port string
}

type p struct {
	External string
	Internal string
}

func readConfig(filename string) (cfg config, err error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	err = json.Unmarshal(b, &cfg)
	if err != nil {
		return
	}
	return
}

func writeConfig(cfg config, filename string) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, b, 0644)
}
