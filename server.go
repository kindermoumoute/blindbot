package main

import (
	"encoding/json"
	"net/http"
	"os"

	"log"

	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/kindermoumoute/blindbot/bot"
)

type FileInfo struct {
	Name  string
	IsDir bool
	Mode  os.FileMode
}

const (
	musicPrefix = "/music/"
	root        = "./music/"
	secretDir   = "/cred"
)

func runServer(b *bot.Bot) {

	r := mux.NewRouter()
	r.HandleFunc("/", playerMainFrame)
	r.HandleFunc("/submit", b.Submit)
	r.HandleFunc(musicPrefix, file)
	r.HandleFunc(musicPrefix+"{path}", file)

	go func() {
		// switch to :http
		if err := http.ListenAndServe(":https", http.HandlerFunc(redirectTLS)); err != nil {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	// remove this
	log.Fatal(http.ListenAndServe(":http", r))

	// uncomment this
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
	// switch to https
	http.Redirect(w, r, "http://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}

func playerMainFrame(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./player.html")
}

func file(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(root, r.URL.Path[len(musicPrefix):])
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
