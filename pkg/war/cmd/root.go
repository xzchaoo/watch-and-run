package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/fatih/color"
	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/xzchaoo/watch-and-run/pkg/war"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

// config file path
var cfgPath string

// watch root dir
var fRoot string

// run commands
var fRun []string

// Auto mode: the idea that convention takes precedence over configuration.
var fAuto bool
var fLogLevel int
var fIgnore []string

// The fDelay parameter is used to implement function debouncing.
// If more than one file change is detected within a short period of time, they will be merged into a single change.
var fDelay time.Duration

// If fCancelLast is true, when a file change is detected, the last ongoing running will be cancelled.
// If fCancelLast is false, it will wait until the last ongoing running process finishes before it starts execution.
var fCancelLast bool

// If the SIGTERM signal fails to stop the run process group within the specified time, then the SIGKILL signal will be sent to the run process group.
// If fTermTimeout is zero, then the SIGKILL signal will be sent directly to the run process group.
var fTermTimeout time.Duration

var rootCmd = &cobra.Command{
	Use: "war",
	Example: `  # auto mode
  # It use current working directory as root if it is not set.
  # It automatically loads the $root/.gitignore as ignore file if it exists.
  # It automatically uses the $root/war_run.sh as run command if it exists and run command is empty.
  # It automatically uses the $root/run.sh as run command if it exists and run command is empty.
  war --auto`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfgPath != "" && len(args) == 1 {
			return errors.New("you cannot use the --config parameter and the config arg at the same time")
		}
		if len(args) == 1 {
			cfgPath = args[0]
		}
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get wd error: %+v", err)
		}

		cfg := war.Config{}
		var cfgDir string
		root := fRoot
		var run []string
		var ignoreLines []string

		if cfgPath != "" {
			if _, err = toml.DecodeFile(cfgPath, &cfg); err != nil {
				return fmt.Errorf("read config file error: %+v", err)
			}
			if cfgDir, err = filepath.Abs(filepath.Dir(cfgPath)); err != nil {
				return fmt.Errorf("get config dir error: %+v", err)
			}
			if root == "" {
				root = cfg.Root
				if root == "" {
					root = wd
				} else if filepath.IsAbs(root) {
					root = cfg.Root
				} else if strings.HasPrefix(root, "wd:") {
					// wd:${relativePath}
					root = filepath.Join(wd, root[len("wd:"):])
				} else if strings.HasPrefix(root, "cfg:") {
					// cfg:${relativePath}
					root = filepath.Join(cfgDir, root[len("cfg:"):])
				} else if strings.HasPrefix(root, "env:") {
					// env:project_root
					root = os.Getenv(root[len("env:"):])
				} else {
					root = filepath.Join(wd, root)
				}
			}
			run = convertToStringSlice(cfg.Run)
			if cfg.IgnoreFile != "" {
				bs, err := os.ReadFile(cfg.IgnoreFile)
				if err != nil {
					return err
				}
				ignoreLines = append(ignoreLines, strings.Split(string(bs), "\n")...)
			}
			ignoreLines = append(ignoreLines, cfg.IgnoreRules...)
		}
		if root == "" {
			root = wd
		}
		log.Println(color.YellowString("root=[%s]", root))

		run = append(run, lo.Map(fRun, func(s string, _ int) string {
			return lo.Ternary(filepath.IsAbs(s), s, filepath.Join(root, s))
		})...)

		if fAuto {
			// auto mode
			{
				path := filepath.Join(root, ".gitignore")
				if bs, err := os.ReadFile(path); err == nil {
					ignoreLines = append(ignoreLines, strings.Split(string(bs), "\n")...)
					log.Println(color.YellowString("[auto] load ignore file from %s", path))
				}
			}
			if len(run) == 0 {
				path := filepath.Join(root, "war_run.sh")
				if _, err := os.Stat(path); err == nil {
					log.Println(color.YellowString("[auto] detect war_run.sh"))
					run = append(run, path)
				}
			}
			if len(run) == 0 {
				path := filepath.Join(root, "run.sh")
				if _, err := os.Stat(path); err == nil {
					log.Println(color.YellowString("[auto] detect run.sh"))
					run = append(run, path)
				}
			}
		}

		if len(fIgnore) > 0 {
			log.Println(color.YellowString("add ignore: %s", fIgnore))
			ignoreLines = append(ignoreLines, fIgnore...)
		}
		if len(run) == 0 {
			return errors.New("run is empty, use -r to specify the run command")
		}
		ignore := gitignore.CompileIgnoreLines(ignoreLines...)
		opts := []war.Option{
			war.WithRoot(root),                   //
			war.WithCfgDir(cfgDir),               //
			war.WithRun(run),                     //
			war.WithIgnore(ignore),               //
			war.WithIncludeExts(cfg.IncludeExts), //
			war.WithEnv(cfg.Env),                 //
			war.WithLogLevel(fLogLevel),          //
		}

		if cmd.Flag("delay").Changed {
			d := war.Duration(fDelay)
			cfg.Delay = &d
		}
		if cmd.Flag("cancel-last").Changed {
			b := fCancelLast
			cfg.CancelLast = &b
		}
		if cmd.Flag("term-timeout").Changed {
			d := war.Duration(fTermTimeout)
			cfg.TermTimeout = &d
		}
		if cfg.Delay != nil {
			opts = append(opts, war.WithDelay(time.Duration(*cfg.Delay)))
		}
		if cfg.CancelLast != nil {
			opts = append(opts, war.WithCancelLast(*cfg.CancelLast))
		}
		if cfg.TermTimeout != nil {
			opts = append(opts, war.WithTermTimeout(time.Duration(*cfg.TermTimeout)))
		}
		w, err := war.NewWatchAndRun(opts...) //
		if err != nil {
			return err
		}
		if err := w.Start(context.Background()); err != nil {
			return err
		}
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		sig := <-sigCh
		log.Printf("receive %s", sig)
		signal.Stop(sigCh)
		return w.Stop(context.Background())
	},
}

func init() {
	rootCmd.AddCommand(exampleCmd)
	rootCmd.Flags().StringVarP(&cfgPath, "config", "c", "", "config file")
	rootCmd.Flags().StringVarP(&fRoot, "root", "", "", "watch root")
	rootCmd.Flags().StringSliceVarP(&fRun, "run", "r", nil, "run cmd")
	rootCmd.Flags().BoolVarP(&fAuto, "auto", "", false, "auto mode")
	rootCmd.Flags().IntVarP(&fLogLevel, "log-level", "l", 1, "log level (0: silent, 1: log file changes, 9: log all)")
	rootCmd.Flags().StringSliceVarP(&fIgnore, "ignore", "i", nil, "ignore pattern")
	rootCmd.Flags().DurationVarP(&fDelay, "delay", "d", time.Second, "run delay")
	rootCmd.Flags().BoolVarP(&fAuto, "cancel-last", "", true, "cancel the last run if it has not already been stopped")
	rootCmd.Flags().DurationVarP(&fTermTimeout, "term-timeout", "", time.Second, "SIGTERM timeout")
}

func Execute() {
	rootCmd.Execute()
}

func convertToStringSlice(a any) []string {
	var ret []string
	switch x := a.(type) {
	case string:
		ret = []string{x}
	case []any:
		ret = lo.Map(x, func(item any, _ int) string {
			return item.(string)
		})
	}
	return ret
}
