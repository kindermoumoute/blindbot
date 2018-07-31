package main

import (
	"encoding/json"
	"net/http"
	"os"

	"log"

	"github.com/gorilla/mux"
	"github.com/kindermoumoute/blindbot/bot"
)

type FileInfo struct {
	Name  string
	IsDir bool
	Mode  os.FileMode
}

const (
	root      = "./music/"
	secretDir = "/cred"
)

func runServer(b *bot.Bot) {

	r := mux.NewRouter()
	r.HandleFunc("/", playerMainFrame)
	r.HandleFunc("/submit", b.Submit)
	r.HandleFunc("/music/{path}", file)

	go func() {
		if err := http.ListenAndServe(":443", http.HandlerFunc(redirectTLS)); err != nil {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()
	log.Fatal(http.ListenAndServe(":80", r))

	//m := &autocert.Manager{
	//	Cache:      autocert.DirCache(secretDir),
	//	Prompt:     autocert.AcceptTOS,
	//	HostPolicy: autocert.HostWhitelist(domain),
	//}
	//
	//ln, err := tls.Listen("tcp", ":https", m.TLSConfig())
	//if err != nil {
	//	log.Fatalf("ssl listener %v", err)
	//}
	//
	//log.Fatal(http.Serve(ln, r))

}

func redirectTLS(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "http://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}

func playerMainFrame(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./player.html")
}

func file(w http.ResponseWriter, r *http.Request) {
	path := root + mux.Vars(r)["path"]
	stat, err := os.Stat(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if stat.IsDir() {
		serveDir(w, r, path)
		return
	}
	http.ServeFile(w, r, path)
}

func serveDir(w http.ResponseWriter, r *http.Request, path string) {
	defer func() {
		if err, ok := recover().(error); ok {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		panic(err)
	}

	files, err := file.Readdir(-1)
	if err != nil {
		panic(err)
	}

	fileinfos := make([]FileInfo, len(files), len(files))

	for i := range files {
		fileinfos[i].Name = files[i].Name()
		fileinfos[i].IsDir = files[i].IsDir()
		fileinfos[i].Mode = files[i].Mode()
	}

	j := json.NewEncoder(w)

	if err := j.Encode(&fileinfos); err != nil {
		panic(err)
	}
}
