package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/djburkhart/gcpgrlx"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fail(err.Error())
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	_ = stderr
	if len(args) < 1 {
		return fmt.Errorf("usage: gcpgrlx <render|render-stdout|render-caddy|plan|validate|init> [flags]")
	}

	switch args[0] {
	case "render":
		return render(args[1:], stdout)
	case "render-stdout":
		return renderStdout(args[1:], stdout)
	case "render-caddy":
		return renderCaddy(args[1:], stdout)
	case "plan":
		return plan(args[1:], stdout)
	case "validate":
		return validate(args[1:], stdout)
	case "init":
		return initConfig(args[1:], stdout)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func render(args []string, stdout io.Writer) error {
	// The CLI is intentionally thin: parse flags, load config, then hand off to the library.
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	configPath := fs.String("f", "", "path to deployment config yaml")
	outDir := fs.String("out", ".", "output directory")
	stdoutOnly := fs.Bool("stdout", false, "write rendered recipe to stdout")
	fs.Parse(args)

	if *configPath == "" {
		return fmt.Errorf("render requires -f <config.yaml>")
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}

	if *stdoutOnly {
		recipe, err := gcpgrlx.RenderRecipe(cfg)
		if err != nil {
			return fmt.Errorf("render recipe: %w", err)
		}
		_, err = fmt.Fprint(stdout, recipe)
		return err
	}

	path, err := gcpgrlx.WriteRecipeFile(cfg, *outDir)
	if err != nil {
		return fmt.Errorf("write recipe: %w", err)
	}

	_, err = fmt.Fprintf(stdout, "wrote %s\n", filepath.Clean(path))
	return err
}

func renderStdout(args []string, stdout io.Writer) error {
	return render(append(args, "--stdout"), stdout)
}

func renderCaddy(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("render-caddy", flag.ExitOnError)
	configPath := fs.String("f", "", "path to deployment config yaml")
	outDir := fs.String("out", ".", "output directory")
	stdoutOnly := fs.Bool("stdout", false, "write rendered Caddyfile to stdout")
	fs.Parse(args)

	if *configPath == "" {
		return fmt.Errorf("render-caddy requires -f <config.yaml>")
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}

	if *stdoutOnly {
		rendered, err := gcpgrlx.RenderCaddyfile(cfg)
		if err != nil {
			return fmt.Errorf("render caddyfile: %w", err)
		}
		_, err = fmt.Fprint(stdout, rendered)
		return err
	}

	path, err := gcpgrlx.WriteCaddyFile(cfg, *outDir)
	if err != nil {
		return fmt.Errorf("write caddyfile: %w", err)
	}

	_, err = fmt.Fprintf(stdout, "wrote %s\n", filepath.Clean(path))
	return err
}

func plan(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("plan", flag.ExitOnError)
	configPath := fs.String("f", "", "path to deployment config yaml")
	fs.Parse(args)

	if *configPath == "" {
		return fmt.Errorf("plan requires -f <config.yaml>")
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}

	rendered, err := gcpgrlx.RenderPlan(cfg)
	if err != nil {
		return fmt.Errorf("render plan: %w", err)
	}

	_, err = fmt.Fprint(stdout, rendered)
	return err
}

func validate(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	configPath := fs.String("f", "", "path to deployment config yaml")
	fs.Parse(args)

	if *configPath == "" {
		return fmt.Errorf("validate requires -f <config.yaml>")
	}

	if _, err := loadConfig(*configPath); err != nil {
		return err
	}

	_, err := fmt.Fprintf(stdout, "config is valid: %s\n", filepath.Clean(*configPath))
	return err
}

func initConfig(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	outPath := fs.String("out", "microservices.yaml", "path to write the sample deployment config")
	stdoutOnly := fs.Bool("stdout", false, "write sample config to stdout")
	fs.Parse(args)

	raw, err := gcpgrlx.SampleConfigYAML()
	if err != nil {
		return fmt.Errorf("render sample config: %w", err)
	}

	if *stdoutOnly {
		_, err = stdout.Write(raw)
		return err
	}

	if err := writeNewFile(*outPath, raw); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "wrote %s\n", filepath.Clean(*outPath))
	return err
}

func loadConfig(path string) (gcpgrlx.Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return gcpgrlx.Config{}, fmt.Errorf("read config: %w", err)
	}

	cfg, err := gcpgrlx.ParseConfig(raw)
	if err != nil {
		return gcpgrlx.Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

func writeNewFile(path string, raw []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("write sample config: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(raw); err != nil {
		return fmt.Errorf("write sample config: %w", err)
	}
	return nil
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
