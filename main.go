package main

import (
	"flag"
	"fmt"
	"github.com/Streamlet/NoteIsSite/config"
	"github.com/Streamlet/NoteIsSite/global"
	"github.com/Streamlet/NoteIsSite/server"
	"github.com/Streamlet/NoteIsSite/util"
	"os"
	"os/signal"
	"syscall"
)

type commandLineArgs struct {
	config string
}

func main() {
	var args commandLineArgs
	flag.StringVar(&args.config, "config", "site.toml", "config file path")
	flag.Parse()

	err := config.LoadSiteConfig(args.config)
	if err != nil {
		fmt.Printf("failed to load %s: %s\n", args.config, err.Error())
		return
	}
	conf := config.GetSiteConfig()

	var srv server.HttpServer
	if conf.Server.Sock != "" {
		srv, err = server.NewSockServer(conf.Server.Sock, conf.Note.NoteRoot, conf.Template.TemplateRoot)
	} else if conf.Server.Port > 0 {
		srv, err = server.NewPortServer(conf.Server.Port, conf.Note.NoteRoot, conf.Template.TemplateRoot)
	} else {
		util.Assert(false, "neither port or sock was specified")
		return
	}

	if err != nil {
		fmt.Printf("failed to init server with node root '%s' and template root '%s': %s.\n", conf.Note.NoteRoot, conf.Template.TemplateRoot, err.Error())
		return
	}

	global.InitErrorChan()

	err = srv.Serve()
	if err != nil {
		fmt.Printf("failed to start server: %s.\n", err.Error())
		return
	}

	fmt.Printf("Server started")
	if conf.Server.Sock != "" {
		fmt.Printf(" on sock '%s'.\n", conf.Server.Sock)
	} else if conf.Server.Port > 0 {
		fmt.Printf(" on port %d.\n", conf.Server.Port)
	}
	fmt.Printf("Note root: %s\n", conf.Note.NoteRoot)
	fmt.Printf("Template root: %s\n", conf.Template.TemplateRoot)
	fmt.Printf("\n")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sig:
			fmt.Printf("Server shutdown.\n")
		case err := <-global.GetErrorChan():
			fmt.Printf("Server error: %s\n", err.Error())
		}
		break
	}

	if err := srv.Shutdown(); err != nil {
		fmt.Printf("Server shutdown: %s\n", err.Error())
	}
}
