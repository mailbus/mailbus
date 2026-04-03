package crypto

import (
	"fmt"
	"os"

	"github.com/mailbus/mailbus/pkg/crypto"
	"github.com/spf13/cobra"
)

var (
	password string
)

func NewEncryptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "encrypt",
		Short: "Encrypt a password for storage in config",
		Long: `Encrypt a password using age encryption.
The encrypted password can be stored in config.yaml under password_encrypted field.

Usage:
  mailbus crypto encrypt --password "mypassword"

The encrypted output will be printed to stdout.`,
		RunE: runEncrypt,
	}

	cmd.Flags().StringVar(&password, "password", "", "Password to encrypt")
	cmd.MarkFlagRequired("password")

	return cmd
}

func runEncrypt(cmd *cobra.Command, args []string) error {
	// Get encryption key from environment
	cryptoKey := os.Getenv("MAILBUS_CRYPTO_KEY")
	if cryptoKey == "" {
		return fmt.Errorf("MAILBUS_CRYPTO_KEY environment variable is not set\n\n" +
			"Please set it first:\n" +
			"  export MAILBUS_CRYPTO_KEY=\"your-encryption-key\"")
	}

	// Encrypt the password
	encrypted, err := crypto.EncryptPassword(password, cryptoKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	// Print the encrypted password (base64 encoded)
	fmt.Printf("password_encrypted: %s\n", encrypted)

	return nil
}
