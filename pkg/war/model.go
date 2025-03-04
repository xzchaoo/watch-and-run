package war

import "time"

type (
	Duration time.Duration
	Config   struct {
		Root string
		// Build string or []string
		Build any
		// Run string or []string
		Run         any
		IncludeExts []string `toml:"include_exts"`
		IgnoreRules []string `toml:"ignore_rules"`
		IgnoreFile  string   `toml:"ignore_file"`
		Delay       *Duration
		CancelLast  *bool             `toml:"cancel_last"`
		TermTimeout *Duration         `toml:"term_timeout"`
		Env         map[string]string `toml:"env"`
	}
	watchedInfo struct {
		file bool
	}
	cancel struct {
		done chan<- struct{}
	}
)

func (d *Duration) UnmarshalText(b []byte) error {
	x, err := time.ParseDuration(string(b))
	if err != nil {
		return err
	}
	*d = Duration(x)
	return nil
}
