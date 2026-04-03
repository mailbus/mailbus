package crypto

import (
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "crypto",
		Short: "Cryptography commands for password encryption",
		Long: `Cryptography commands for securing passwords in MailBus configuration.

These commands allow you to encrypt passwords for storage in config.yaml
instead of using plain environment variables.`,
	}

	// Add subcommands
	cmd.AddCommand(NewEncryptCmd())

	return cmd
}
