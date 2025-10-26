package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(newInitCmd())
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Config initialization",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Help()
		},
	}

	cmd.AddCommand(
		newInitProfileCmd(),
		newInitConfigCmd(),
	)
	return cmd
}

func newInitProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Initialize aws profile file",
		RunE:  runInitProfileCmd,
	}

	return cmd
}

func runInitProfileCmd(_ *cobra.Command, _ []string) error {
	usr, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "Cant find HOME path:")
	}

	profileDir := filepath.Join(usr, ".aws")
	if _, err := os.Stat(profileDir); os.IsNotExist(err) {
		err = os.MkdirAll(profileDir, 0o755)
		if err != nil {
			return errors.Wrap(err, "Error creating .aws dir")
		}
	} else {
		return errors.New(".aws directory already exists")
	}

	defaultProfile := `[default]
aws_access_key_id = XXXXX 
aws_secret_access_key = XXXXX
`
	credentialsFile := filepath.Join(profileDir, "credentials")

	if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
		err = os.WriteFile(credentialsFile, []byte(defaultProfile), 0o644)
		if err != nil {
			return errors.Wrap(err, "Error writing credentials file")
		}
		fmt.Printf("Created credentials file at %s\n", credentialsFile)
	} else {
		return errors.New("Profile already exists at .aws/credentials")
	}

	return nil
}

func newInitConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Initialize myaws config",
		RunE:  runInitConfigCmd,
	}

	return cmd
}

func runInitConfigCmd(_ *cobra.Command, _ []string) error {
	usr, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "Cant find HOME path:")
	}

	configFile := filepath.Join(usr, ".myaws.yml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		defaultConfig := `profile: default
region: us-east-1
`
		err = os.WriteFile(configFile, []byte(defaultConfig), 0o644)
		if err != nil {
			return errors.Wrap(err, "Cant find HOME path:")
		}
		fmt.Printf("Created config file => (%s/.myaws.yml)", usr)
	} else {
		return errors.New(".myaws.yml config file already exists")
	}

	return nil
}
