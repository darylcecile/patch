package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/darylcecile/patch/internal/app"
)

func main() {
	flag.Parse()
	root := ""
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}
	if flag.NArg() > 1 {
		log.Fatal("usage: patch [workspace-directory]")
	}
	var err error
	if root != "" {
		root, err = filepath.Abs(root)
		if err != nil {
			log.Fatal(err)
		}
	}
	home := os.Getenv("PATCH_HOME")
	if home == "" {
		home, err = os.UserConfigDir()
		if err != nil {
			log.Fatal(err)
		}
		home = filepath.Join(home, "Patch")
	}
	home, err = filepath.Abs(home)
	if err != nil {
		log.Fatal(err)
	}
	if err := app.Run(root, home, flag.NArg() > 0); err != nil {
		log.Fatal(err)
	}
}
