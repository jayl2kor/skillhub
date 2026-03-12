package cli

import (
	"testing"
)

func TestRootCommandExists(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	if rootCmd.Use != "skillhub" {
		t.Errorf("expected Use 'skillhub', got %q", rootCmd.Use)
	}
}

func TestSubcommandRegistration(t *testing.T) {
	expected := []string{
		"init", "list", "doctor", "repo", "search",
		"info", "install", "run", "update", "remove",
		"cache", "create", "lint", "package", "completion", "verify",
	}

	commands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		commands[cmd.Name()] = true
	}

	for _, name := range expected {
		if !commands[name] {
			t.Errorf("command %q not registered", name)
		}
	}
}

func TestRepoSubcommands(t *testing.T) {
	expected := []string{"add", "list", "remove", "index"}

	commands := make(map[string]bool)
	for _, cmd := range repoCmd.Commands() {
		commands[cmd.Name()] = true
	}

	for _, name := range expected {
		if !commands[name] {
			t.Errorf("repo subcommand %q not registered", name)
		}
	}
}
