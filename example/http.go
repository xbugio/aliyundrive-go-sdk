package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/xbugio/aliyundrive-go-sdk"
	alifs "github.com/xbugio/aliyundrive-go-sdk/fs"
)

func main() {
	var (
		refreshToken string
		addr         string
		root         string
	)

	flag.StringVar(&refreshToken, "refresh-token", "", "refresh token")
	flag.StringVar(&addr, "addr", "", "listen address")
	flag.StringVar(&root, "root", "/", "root")
	flag.Parse()

	c := &aliyundrive.Drive{
		RefreshToken: refreshToken,
	}
	if err := c.Init(); err != nil {
		log.Fatal(err)
	}
	defer c.Destory()

	fsys := alifs.New(c, root)

	handler := http.FileServer(http.FS(fsys))
	server := &http.Server{
		Addr:     addr,
		Handler:  handler,
		ErrorLog: nil,
	}
	server.ListenAndServe()
}
