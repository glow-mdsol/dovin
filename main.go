package main

import (
	_ "embed"
	"log"
	"os"
	"strconv"

	"fyne.io/systray"
	"github.com/glow-mdsol/dovin/api"
	"github.com/glow-mdsol/dovin/scheduler"
	"github.com/glow-mdsol/dovin/store"
	"github.com/glow-mdsol/dovin/ui"
)

//go:embed assets/icon.png
var iconData []byte

func main() {
	logFile := os.Getenv("HOME") + "/Library/Logs/dovin.log"
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		log.SetOutput(f)
		defer f.Close()
	}

	s, err := store.Open()
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer s.Close()

	srv, err := api.New(s)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	port := srv.Port()
	if err := s.ConfigSet("port", strconv.Itoa(port)); err != nil {
		log.Printf("save port: %v", err)
	}

	go func() {
		if err := srv.Serve(srv.Handler(ui.FileSystem())); err != nil {
			log.Printf("server: %v", err)
		}
	}()

	done := make(chan struct{})
	go scheduler.Run(s, done)

	systray.Run(func() {
		systray.SetIcon(iconData)
		systray.SetTooltip("Dovin")
		updateBadge(s)

		mOpen := systray.AddMenuItem("Open", "Open Dovin")
		mAdd := systray.AddMenuItem("Add Task…", "Add a new task")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit Dovin")

		go func() {
			for {
				select {
				case <-mOpen.ClickedCh:
					openWebview(port, "")
					updateBadge(s)
				case <-mAdd.ClickedCh:
					openWebview(port, "#add")
					updateBadge(s)
				case <-mQuit.ClickedCh:
					close(done)
					systray.Quit()
				}
			}
		}()
	}, func() {})
}

func updateBadge(s *store.Store) {
	n, err := s.PendingTaskCount()
	if err != nil || n == 0 {
		systray.SetTitle("")
		return
	}
	systray.SetTitle(strconv.Itoa(n))
}
