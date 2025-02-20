package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	green = color.New(color.FgGreen).SprintFunc()
	red   = color.New(color.FgRed).SprintFunc()

	profile string // for AWS only
	project string // for GCP only
	stdout  bool
	stderr  bool
	idFile  string
	mtx     sync.Mutex
	cs      map[string]*exec.Cmd

	rootCmd = &cobra.Command{
		Use:   "sweeper",
		Short: "A simple internal, enterprise scraping and organizing tool for AI",
		Long: `A simple internal, enterprise scraping and organizing tool for AI.

[version=` + version + `, commit=` + commit + `]`,
		Run:          run,
		SilenceUsage: true,
	}
)

func main() {
	go func() {
		s := make(chan os.Signal, 1)
		signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
		sig := fmt.Errorf("%s", <-s)
		_ = sig

		for _, c := range cs {
			err := c.Process.Signal(syscall.SIGTERM)
			if err != nil {
				info("failed to terminate process, force kill...")
				_ = c.Process.Signal(syscall.SIGKILL)
			}
		}

		os.Exit(0)
	}()

	log.SetOutput(os.Stdout) // for easy grep
	rootCmd.Flags().SortFlags = false
	rootCmd.Flags().StringVar(&idFile, "id-file", "", "identity file, input to -i in ssh (AWS only)")
	rootCmd.Flags().BoolVar(&stdout, "stdout", true, "print stdout output")
	rootCmd.Flags().BoolVar(&stderr, "stderr", true, "print stderr output")
	rootCmd.Flags().StringVar(&profile, "profile", "", "AWS profile, valid only if 'asg', optional")
	rootCmd.Flags().StringVar(&project, "project", "", "GCP project, valid only if 'mig', optional")
	rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) {
	if len(args) < 3 {
		fail("invalid arguments, see -h")
		return
	}
}

func info(v ...interface{}) {
	m := fmt.Sprintln(v...)
	log.Printf("%s %s", green("[info]"), m)
}

func fail(v ...interface{}) {
	m := fmt.Sprintln(v...)
	log.Printf("%s %s", red("[error]"), m)
}

func failx(v ...interface{}) {
	fail(v...)
	os.Exit(1)
}
