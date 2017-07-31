package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}
	chat := os.Getenv("CHAT")
	if chat == "" {
		log.Fatal("$CHAT must be set")
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	index, err := indexHandler(wd)
	if err != nil {
		log.Fatal(err)
	}
	web := http.FileServer(http.Dir(wd + "/web"))

	http.Handle("/", index)
	http.Handle("/web/", http.StripPrefix("/web/", web))
	http.Handle("/ws", wsHandler(chat))

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func wsHandler(upstream string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer, err := net.Dial("tcp", upstream)
		if err != nil {
			w.WriteHeader(502)
			return
		}
		if err := r.Write(peer); err != nil {
			w.WriteHeader(502)
			return
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			w.WriteHeader(500)
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			w.WriteHeader(500)
			return
		}

		log.Printf(
			"serving %s < %s <~> %s > %s",
			peer.RemoteAddr(), peer.LocalAddr(), conn.RemoteAddr(), conn.LocalAddr(),
		)

		go func() {
			defer peer.Close()
			defer conn.Close()
			io.Copy(peer, conn)
		}()
		go func() {
			defer peer.Close()
			defer conn.Close()
			io.Copy(conn, peer)
		}()
	})
}

func indexHandler(wd string) (http.Handler, error) {
	index, err := os.Open(wd + "/web/index.html")
	if err != nil {
		return nil, err
	}
	stat, err := index.Stat()
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "", stat.ModTime(), index)
	}), nil
}
