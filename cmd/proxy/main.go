package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	source := flag.String("source", "", "Source url as http://example.com:8080/")
	listen := flag.String("listen", ":9001", "Listen 127.0.0.1:8000")
	endpoint := flag.String("endpoint", "/", "Endpoint")
	help := flag.Bool("h", false, "Display Help.")
	flag.Parse()

	if *help {
		fmt.Println("Simple Reverse proxy.")
		fmt.Println()
		flag.PrintDefaults()
		return
	}

	origin, err := url.Parse(*source)
	if err != nil {
		log.Fatal(err)
	}

	director := func(req *http.Request) {
		req.Header.Add("X-Forwarded-Host", req.Host)
		req.Header.Add("X-Origin-Host", origin.Host)
		req.URL.Scheme = origin.Scheme
		req.URL.Host = origin.Host
	}

	proxy := &httputil.ReverseProxy{Director: director}

	http.HandleFunc(*endpoint, func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})

	log.Fatal(http.ListenAndServe(*listen, nil))
}
