package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/djburkhart/gcpgrlx"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: gcpgrlx render -f <config.yaml> [-out <dir>]")
	}

	switch os.Args[1] {
	case "render":
		render(os.Args[2:])
	default:
		fail("unknown command: " + os.Args[1])
	}
}

func render(args []string) {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	configPath := fs.String("f", "", "path to deployment config yaml")
	outDir := fs.String("out", ".", "output directory")
	stdout := fs.Bool("stdout", false, "write rendered recipe to stdout")
	fs.Parse(args)

	if *configPath == "" {
		fail("render requires -f <config.yaml>")
	}

	raw, err := os.ReadFile(*configPath)
	if err != nil {
		fail("read config: " + err.Error())
	}

	cfg, err := gcpgrlx.ParseConfig(raw)
	if err != nil {
		fail("parse config: " + err.Error())
	}

	if *stdout {
		recipe, err := gcpgrlx.RenderRecipe(cfg)
		if err != nil {
			fail("render recipe: " + err.Error())
		}
		fmt.Print(recipe)
		return
	}

	path, err := gcpgrlx.WriteRecipeFile(cfg, *outDir)
	if err != nil {
		fail("write recipe: " + err.Error())
	}

	fmt.Printf("wrote %s\n", filepath.Clean(path))
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
