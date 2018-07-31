package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kindermoumoute/blindbot/bot"
)

type FileInfo struct {
	Name  string
	IsDir bool
	Mode  os.FileMode
}

const (
	filePrefix   = "/music/"
	submitPrefix = "/submit/"
	root         = "./music"
)

func runServer(b *bot.Bot) {
	http.HandleFunc("/", playerMainFrame)
	http.HandleFunc(submitPrefix, b.Submit)
	http.HandleFunc(filePrefix, file)
	go func() {
		if err := http.ListenAndServe(":80", http.HandlerFunc(redirectTLS)); err != nil {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()
	http.ListenAndServeTLS(":443", "cred/server.crt", "cred/server.key", nil)
}

func redirectTLS(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}

func playerMainFrame(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./player.html")
}

func file(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(root, r.URL.Path[len(filePrefix):])
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
