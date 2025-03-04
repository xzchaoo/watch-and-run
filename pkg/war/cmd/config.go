package cmd

import (
	_ "embed"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
)

//go:embed example.toml
var exampleConfigBytes []byte

var exampleCmd = &cobra.Command{
	Use:   "example [/path/to/example.war.toml]",
	Short: "Generate example config",
	Example: `  # Generate example config to STDOUT
  war example

  # Generate example config to file
  war example /path/to/example.war.toml  
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			os.Stdout.Write(exampleConfigBytes)
			return nil
		}
		path := args[0]
		if _, err := os.Stat(path); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		} else {
			return fmt.Errorf("file already exists: %s", path)
		}
		if err := os.WriteFile(path, exampleConfigBytes, 0644); err != nil {
			return err
		}
		log.Printf("Generate demo config to %s", path)
		return nil
	},
}
