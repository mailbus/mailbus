package mark

import (
	"context"
	"fmt"
	"time"

	"github.com/mailbus/mailbus/pkg/adapter"
	"github.com/mailbus/mailbus/pkg/config"
	"github.com/spf13/cobra"
)

var (
	messageID  string
	markAction string
	targetFolder string
	accountName string
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mark",
		Short: "Mark messages with actions",
		Long: `Mark a message with an action such as read, unread, delete, or move.

This command modifies the status or location of a message on the IMAP server.`,
		Example: `  # Mark a message as read
  mailbus mark --id "123@localhost" --action read

  # Mark a message as deleted
  mailbus mark --id "123@localhost" --action delete

  # Move a message to another folder
  mailbus mark --id "123@localhost" --action move --folder "Processed"`,
		RunE: runMark,
	}

	cmd.Flags().StringVarP(&messageID, "id", "i", "", "message ID (required)")
	cmd.Flags().StringVarP(&markAction, "action", "a", "", "action: read, unread, delete, undelete, flag, unflag, move (required)")
	cmd.Flags().StringVarP(&targetFolder, "folder", "F", "", "target folder (required for move action)")
	cmd.Flags().StringVarP(&accountName, "account", "A", "", "account to use")

	cmd.MarkFlagRequired("id")
	cmd.MarkFlagRequired("action")

	return cmd
}

func runMark(cmd *cobra.Command, args []string) error {
	// Validate action
	if markAction == "move" && targetFolder == "" {
		return fmt.Errorf("--folder is required for move action")
	}

	// Load config
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get account
	account, err := getAccount(cfg, accountName)
	if err != nil {
		return err
	}

	// Get password
	password, err := account.GetPassword()
	if err != nil {
		return err
	}

	// Connect to IMAP server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	imapAdapter := adapter.NewIMAPAdapter()
	connCfg := &adapter.ConnectionConfig{
		Host:     account.IMAP.Host,
		Port:     account.IMAP.Port,
		Username: account.Username,
		Password: password,
		UseTLS:   account.IMAP.UseTLS,
	}

	if err := imapAdapter.Connect(ctx, connCfg); err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer imapAdapter.Close()

	// Perform action
	switch markAction {
	case "move":
		// Move message to folder
		if err := imapAdapter.MoveMessage(messageID, targetFolder); err != nil {
			return fmt.Errorf("failed to move message: %w", err)
		}
		fmt.Printf("Message moved to %s\n", targetFolder)

	default:
		// Convert action string to MarkAction
		action, err := parseMarkAction(markAction)
		if err != nil {
			return err
		}

		// Mark message
		if err := imapAdapter.Mark(ctx, messageID, action); err != nil {
			return fmt.Errorf("failed to mark message: %w", err)
		}
		fmt.Printf("Message marked as %s\n", markAction)
	}

	return nil
}

func parseMarkAction(action string) (adapter.MarkAction, error) {
	switch action {
	case "read":
		return adapter.MarkActionSeen, nil
	case "unread":
		return adapter.MarkActionUnseen, nil
	case "delete":
		return adapter.MarkActionDelete, nil
	case "undelete":
		return adapter.MarkActionUndelete, nil
	case "flag":
		return adapter.MarkActionFlag, nil
	case "unflag":
		return adapter.MarkActionUnflag, nil
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

// Helper functions
func loadConfig() (*config.Config, error) {
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		var err error
		configPath, err = config.ConfigPath()
		if err != nil {
			return nil, err
		}
	}
	return config.LoadConfig(configPath)
}

func getAccount(cfg *config.Config, name string) (*config.AccountConfig, error) {
	if name == "" {
		name = cfg.DefaultAccount
	}
	return cfg.GetAccount(name)
}

// cmd variable for flag access
var cmd = &cobra.Command{}
