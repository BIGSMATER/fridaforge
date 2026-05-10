package main

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
	if rootCmd.Use != "fridaforge" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "fridaforge")
	}
	if rootCmd.Version != version {
		t.Errorf("rootCmd.Version = %q, want %q", rootCmd.Version, version)
	}
	if !rootCmd.SilenceErrors {
		t.Error("rootCmd.SilenceErrors should be true")
	}
	if !rootCmd.SilenceUsage {
		t.Error("rootCmd.SilenceUsage should be true")
	}
	if rootCmd.PersistentPreRun == nil {
		t.Fatal("rootCmd.PersistentPreRun should not be nil")
	}
}

func TestSubCommands(t *testing.T) {
	children := rootCmd.Commands()

	find := func(name string) bool {
		for _, c := range children {
			if c.Name() == name {
				return true
			}
		}
		return false
	}

	if !find("spec") {
		t.Error("rootCmd should have 'spec' subcommand")
	}
	if !find("device") {
		t.Error("rootCmd should have 'device' subcommand")
	}

	specChildren := findSubCommand(children, "spec").Commands()
	if !findSub(specChildren, "validate") {
		t.Error("spec should have 'validate' subcommand")
	}

	deviceChildren := findSubCommand(children, "device").Commands()
	if !findSub(deviceChildren, "list") {
		t.Error("device should have 'list' subcommand")
	}
}

func TestCheckEthicalDisclaimerAlreadyAgreed(t *testing.T) {
	if err := checkEthicalDisclaimer(); err != nil {
		t.Fatalf("checkEthicalDisclaimer() returned error: %v", err)
	}
}

func findSubCommand(cmds []*cobra.Command, name string) *cobra.Command {
	for _, c := range cmds {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func findSub(cmds []*cobra.Command, name string) bool {
	return findSubCommand(cmds, name) != nil
}

func TestMain(m *testing.M) {
	os.Setenv("HOME", os.Getenv("HOME"))
	os.Exit(m.Run())
}
