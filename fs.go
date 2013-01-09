package main

import (
	"github.com/howeyc/fsnotify"
	"path/filepath"
	"log"
	"os"
	"os/exec"
	"bytes"
)

func notifyBrowsers() {
	reloadCfg.mu.Lock()
	defer reloadCfg.mu.Unlock()
	for _, c := range reloadCfg.clients {
		defer c.conn.Close()
		reload := "HTTP/1.1 200 OK\r\n"
		reload += "Cache-Control: no-cache\r\nContent-Type: text/javascript\r\n\r\n"
		reload += "location.reload(true);"
		c.buf.Write([]byte(reload))
		c.buf.Flush()
	}
	reloadCfg.clients = make([]Client, 0)
}

func processFsEvents() {
	for {
		events := <-reloadCfg.eventsCh
		command := reloadCfg.command
		if command != "" {
			args := make([]string, len(events)*2)
			for i, e := range events {
				args[2*i] = e.Event
				args[2*i + 1] = e.File
			}
			sub := exec.Command(command, args...)
			var out bytes.Buffer
			sub.Stdout = &out
			err := sub.Run()
			if err == nil {
				log.Println("run " + command + " ok; output: ", out.String())
				notifyBrowsers()
			} else {
				log.Println("ERROR running " + command, err)
			}
		} else {
			notifyBrowsers()
		}
	}
}

func startFsMonitor() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	walkFn := func(path string, info os.FileInfo, err error) error {
		// TODO handle error, like permission denied
		if info.IsDir() {
			if shouldIgnore(path) {
				log.Println("ignore--------", path)
				return filepath.SkipDir
			}
			log.Println("add", path)
			if e := watcher.Watch(path); e != nil {
				log.Println("WARN", e)
			}
		}
		return nil
	}

	go func() {
		for {
			select {
			case ev := <-watcher.Event:
	            // 4 events are reported: RENAME | CREATE when editing this file
				log.Println("event:", ev)
			case err := <-watcher.Error:
				log.Println("error:", err)
			}
		}
	}()

	root, _ := filepath.Abs(reloadCfg.root)
	if err := filepath.Walk(root, walkFn); err != nil {
		log.Println(err)
	}

}


