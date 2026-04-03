package main

import (
	"fmt"
	"os"

	"github.com/mailbus/mailbus/cmd/config"
	"github.com/mailbus/mailbus/cmd/crypto"
	"github.com/mailbus/mailbus/cmd/list"
	"github.com/mailbus/mailbus/cmd/mark"
	"github.com/mailbus/mailbus/cmd/poll"
	"github.com/mailbus/mailbus/cmd/send"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var verbose bool

	rootCmd := &cobra.Command{
		Use:   "mailbus",
		Short: "Email-based message bus for agent communication",
		Long: `MailBus is a CLI tool that turns email into a message bus for agent communication.
It enables any script or program to communicate asynchronously using standard email protocols (SMTP/IMAP).`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if verbose {
				// Enable verbose logging
				os.Setenv("MAILBUS_VERBOSE", "1")
			}
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringP("account", "A", "", "account to use (default: default_account from config)")
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path (default: ~/.mailbus/config.yaml)")

	// Add subcommands
	rootCmd.AddCommand(send.NewCmd())
	rootCmd.AddCommand(poll.NewCmd())
	rootCmd.AddCommand(list.NewCmd())
	rootCmd.AddCommand(mark.NewCmd())
	rootCmd.AddCommand(config.NewCmd())
	rootCmd.AddCommand(crypto.NewCmd())

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
