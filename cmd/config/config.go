package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mailbus/mailbus/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	initForce bool
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage mailbus configuration",
		Long: `Manage the mailbus configuration file.

Configuration is stored in ~/.mailbus/config.yaml by default.`,
	}

	// Add subcommands
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newSetCmd())

	return cmd
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration file",
		Long: `Create a new configuration file in ~/.mailbus/config.yaml.

If the file already exists, use --force to overwrite it.`,
		Example: `  # Initialize configuration
  mailbus config init

  # Force overwrite existing configuration
  mailbus config init --force`,
		RunE: runInit,
	}

	cmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing configuration")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	// Get config path
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(configPath); err == nil && !initForce {
		return fmt.Errorf("config file already exists at %s (use --force to overwrite)", configPath)
	}

	// Create config directory
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default config
	cfg := config.DefaultConfig()

	// Add example account
	cfg.Accounts["default"] = &config.AccountConfig{}
	cfg.Accounts["default"].Name = "default"
	cfg.Accounts["default"].IMAP.Host = "imap.gmail.com"
	cfg.Accounts["default"].IMAP.Port = 993
	cfg.Accounts["default"].IMAP.UseTLS = true
	cfg.Accounts["default"].SMTP.Host = "smtp.gmail.com"
	cfg.Accounts["default"].SMTP.Port = 587
	cfg.Accounts["default"].SMTP.UseTLS = true
	cfg.Accounts["default"].Username = "your-email@gmail.com"
	cfg.Accounts["default"].PasswordEnv = "MAILBUS_PASSWORD"
	cfg.Accounts["default"].From = "your-email@gmail.com"

	// Save config
	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Configuration created at: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit the configuration file with your email account details")
	fmt.Println("2. Set the password environment variable:")
	fmt.Printf("   export %s=your-password\n", cfg.Accounts["default"].PasswordEnv)
	fmt.Println("3. Validate the configuration:")
	fmt.Println("   mailbus config validate")

	return nil
}

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long: `Check if the configuration file is valid and can be loaded.`,
		Example: `  mailbus config validate`,
		RunE: runValidate,
	}

	return cmd
}

func runValidate(cmd *cobra.Command, args []string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found at %s (run 'mailbus config init' to create it)", configPath)
	}

	// Load and validate config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Check password environment variables
	fmt.Println("Configuration is valid!")
	fmt.Printf("\nAccounts: %d\n", len(cfg.Accounts))
	fmt.Printf("Default account: %s\n", cfg.DefaultAccount)

	for name, account := range cfg.Accounts {
		fmt.Printf("\n  %s:\n", name)
		fmt.Printf("    Username: %s\n", account.Username)
		fmt.Printf("    Password env: %s", account.PasswordEnv)

		// Check if password is set
		password := os.Getenv(account.PasswordEnv)
		if password == "" {
			fmt.Printf(" (not set)")
		} else {
			fmt.Printf(" (set)")
		}
		fmt.Println()
	}

	return nil
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured accounts",
		Long: `Display all accounts configured in the configuration file.`,
		Example: `  mailbus config list`,
		RunE: runList,
	}

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Config file: %s\n\n", configPath)
	fmt.Printf("Default account: %s\n\n", cfg.DefaultAccount)

	if len(cfg.Accounts) == 0 {
		fmt.Println("No accounts configured")
		return nil
	}

	fmt.Println("Accounts:")
	for name, account := range cfg.Accounts {
		marker := " "
		if name == cfg.DefaultAccount {
			marker = "*"
		}
		fmt.Printf("  %s %s\n", marker, name)
		fmt.Printf("      Username: %s\n", account.Username)
		fmt.Printf("      From: %s\n", account.From)
		fmt.Printf("      IMAP: %s:%d (TLS: %v)\n", account.IMAP.Host, account.IMAP.Port, account.IMAP.UseTLS)
		fmt.Printf("      SMTP: %s:%d (TLS: %v)\n", account.SMTP.Host, account.SMTP.Port, account.SMTP.UseTLS)
	}

	return nil
}

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new account",
		Long: `Add a new email account to the configuration.`,
		Example: `  mailbus config add --name work --username work@example.com`,
		RunE: runAdd,
	}

	// Account flags
	var name, username, passwordEnv, from string
	var imapHost, smtpHost string
	var imapPort, smtpPort int
	var imapTLS, smtpTLS bool

	cmd.Flags().StringVar(&name, "name", "", "account name (required)")
	cmd.Flags().StringVar(&username, "username", "", "email username")
	cmd.Flags().StringVar(&passwordEnv, "password-env", "", "environment variable name for password")
	cmd.Flags().StringVar(&from, "from", "", "default from address")
	cmd.Flags().StringVar(&imapHost, "imap-host", "", "IMAP server host")
	cmd.Flags().IntVar(&imapPort, "imap-port", 993, "IMAP server port")
	cmd.Flags().BoolVar(&imapTLS, "imap-tls", true, "IMAP use TLS")
	cmd.Flags().StringVar(&smtpHost, "smtp-host", "", "SMTP server host")
	cmd.Flags().IntVar(&smtpPort, "smtp-port", 587, "SMTP server port")
	cmd.Flags().BoolVar(&smtpTLS, "smtp-tls", true, "SMTP use TLS")

	cmd.MarkFlagRequired("name")

	return cmd
}

func runAdd(cmd *cobra.Command, args []string) error {
	// This is a simplified version - in a real implementation,
	// you'd use viper to bind flags and then read them
	// For now, just show a message
	fmt.Println("Add account command - use with flags:")
	fmt.Println("  --name: account name (required)")
	fmt.Println("  --username: email username")
	fmt.Println("  --password-env: environment variable for password")
	fmt.Println("  --from: default from address")
	fmt.Println("  --imap-host/--imap-port/--imap-tls: IMAP settings")
	fmt.Println("  --smtp-host/--smtp-port/--smtp-tls: SMTP settings")

	return fmt.Errorf("not yet implemented")
}

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove an account",
		Long: `Remove an email account from the configuration.`,
		Example: `  mailbus config remove --name work`,
		RunE: runRemove,
	}

	var name string
	cmd.Flags().StringVar(&name, "name", "", "account name (required)")
	cmd.MarkFlagRequired("name")

	return cmd
}

func runRemove(cmd *cobra.Command, args []string) error {
	// TODO: Implement account removal
	return fmt.Errorf("not yet implemented")
}

func newSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set global configuration options",
		Long: `Set global configuration options like poll interval, timeout, etc.`,
		Example: `  mailbus config set --poll-interval 60
  mailbus config set --verbose`,
		RunE: runSet,
	}

	var pollInterval, timeout, handlerTimeout int
	var verbose bool

	cmd.Flags().IntVar(&pollInterval, "poll-interval", 0, "poll interval in seconds")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "operation timeout in seconds")
	cmd.Flags().IntVar(&handlerTimeout, "handler-timeout", 0, "handler timeout in seconds")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "enable verbose output")

	return cmd
}

func runSet(cmd *cobra.Command, args []string) error {
	// TODO: Implement setting global options
	return fmt.Errorf("not yet implemented")
}

func getConfigPath() (string, error) {
	// Check if config flag is set
	if configPath := viper.GetString("config"); configPath != "" {
		return configPath, nil
	}
	return config.ConfigPath()
}

// cmd variable for flag access
var cmd = &cobra.Command{}
