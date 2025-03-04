package war

import (
	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/samber/lo"
	"time"
)

type (
	options struct {
		root        string
		cfgDir      string
		run         []string
		includeExts map[string]struct{}
		ignore      *gitignore.GitIgnore
		cancelLast  bool
		delay       time.Duration
		termTimeout time.Duration
		env         map[string]string
		logLevel    int
	}
	Option func(*options)
)

func WithRoot(root string) Option {
	return func(o *options) {
		o.root = root
	}
}

func WithCfgDir(cfgDir string) Option {
	return func(o *options) {
		o.cfgDir = cfgDir
	}
}

func WithRun(run []string) Option {
	return func(o *options) {
		o.run = run
	}
}

func WithIncludeExts(exts []string) Option {
	return func(o *options) {
		o.includeExts = lo.SliceToMap(exts, func(item string) (string, struct{}) {
			return item, struct{}{}
		})
	}
}

func WithIgnore(ignore *gitignore.GitIgnore) Option {
	return func(o *options) {
		o.ignore = ignore
	}
}

func WithCancelLast(b bool) Option {
	return func(o *options) {
		o.cancelLast = b
	}
}

func WithDelay(delay time.Duration) Option {
	return func(o *options) {
		o.delay = delay
	}
}

func WithTermTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.termTimeout = timeout
	}
}

func WithEnv(env map[string]string) Option {
	return func(o *options) {
		o.env = env
	}
}

func WithLogLevel(logLevel int) Option {
	return func(o *options) {
		o.logLevel = logLevel
	}
}
