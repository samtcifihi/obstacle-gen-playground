// Command obstacle-gen-playground serves a browser app for generating
// random obstacles on game boards.
package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "address to listen on")
	open := flag.Bool("open", false, "open the app in the default browser once the server is listening")
	flag.Parse()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	url := browserURL(ln.Addr())
	log.Printf("Listening on %s", url)
	if *open {
		openBrowser(url)
	}
	log.Fatal(http.Serve(ln, newHandler()))
}

// browserURL returns a URL a local browser can use to reach addr.
func browserURL(addr net.Addr) string {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "http://" + addr.String() + "/"
	}
	if ip := net.ParseIP(host); ip == nil || ip.IsUnspecified() || ip.IsLoopback() {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port) + "/"
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	go func() {
		if err := cmd.Run(); err != nil {
			log.Printf("Couldn't open a browser (%v); visit %s manually", err, url)
		}
	}()
}
