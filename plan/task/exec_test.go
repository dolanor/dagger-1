package task_test

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	logg("proxied tunneling")
	dest_conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}
func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}
func handleHTTP(w http.ResponseWriter, req *http.Request) {
	logg("proxied http")
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
func runServer() {
	//var pemPath string
	//flag.StringVar(&pemPath, "pem", "server.pem", "path to pem file")
	//var keyPath string
	//flag.StringVar(&keyPath, "key", "server.key", "path to key file")
	//var proto string
	//flag.StringVar(&proto, "proto", "https", "Proxy protocol (http or https)")
	//flag.Parse()
	proto := "http"
	if proto != "http" && proto != "https" {
		log.Fatal("Protocol must be either http or https")
	}
	server := &http.Server{
		Addr: ":8888",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				handleTunneling(w, r)
			} else {
				handleHTTP(w, r)
			}
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	if proto == "http" {
		log.Fatal(server.ListenAndServe())
	} else {
		//log.Fatal(server.ListenAndServeTLS(pemPath, keyPath))
	}
}

var flog *os.File

func init() {
	var err error
	flog, err = os.Create("/tmp/dagger-task-test.log")
	if err != nil {
		panic(err)
	}

}

func logg(msg string, args ...interface{}) {
	fmt.Fprintf(flog, msg, args...)
}

func TestProxy(t *testing.T) {
	go runServer()

	cmd := exec.Command("dagger", []string{"do", "--no-cache", "-p", "testdata/proxy.cue", "display"}...)

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "http_proxy=http://localhost:8887")
	cmd.Env = append(cmd.Env, "HTTP_PROXY=http://localhost:8887")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
}
