package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/xbugio/aliyundrive-go-sdk"
	alifs "github.com/xbugio/aliyundrive-go-sdk/fs"
)

func main() {
	var (
		userId       string
		driveId      string
		deviceId     string
		refreshToken string
		addr         string
		root         string
	)

	flag.StringVar(&userId, "user-id", "", "user id")
	flag.StringVar(&driveId, "drive-id", "", "drive id")
	flag.StringVar(&deviceId, "device-id", "", "device id (cna)")
	flag.StringVar(&refreshToken, "refresh-token", "", "refresh token")
	flag.StringVar(&addr, "addr", "", "listen address")
	flag.StringVar(&root, "root", "/", "root")
	flag.Parse()

	c := aliyundrive.New()
	refreshTokenManager := aliyundrive.NewRefreshTokenManager(c, refreshToken)
	keepaliveTokenManager := aliyundrive.NewKeepAliveTokenManager(refreshTokenManager)
	signatureManager := aliyundrive.NewSignatureManager(c, deviceId, userId, nil)
	c.SetOption(aliyundrive.WithDriveId(driveId))
	c.SetOption(aliyundrive.WithDeviceId(deviceId))
	c.SetOption(aliyundrive.WithTokenManager(keepaliveTokenManager))
	c.SetOption(aliyundrive.WithSignatureManager(signatureManager))
	fsys := alifs.New(c, root)

	keepaliveTokenManager.KeepAlive(context.Background(), time.Second*5)

	handler := http.FileServer(http.FS(fsys))
	server := &http.Server{
		Addr:     addr,
		Handler:  handler,
		ErrorLog: log.New(os.Stderr, "aliyundrive-go-sdk", log.LstdFlags),
	}
	server.ListenAndServe()
	keepaliveTokenManager.WaitStop()
}
