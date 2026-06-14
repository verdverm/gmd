package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var llmProfileShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show role->provider mappings for a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}
		name := args[0]
		profile, ok := cfg.LLM.Profiles[name]
		if !ok {
			return fmt.Errorf("profile %q not found", name)
		}
		fmt.Printf("Profile: %s\n", name)
		printRole("embedding", profile.Embedding)
		printRole("expansion", profile.Expansion)
		printRole("rerank", profile.Rerank)
		printRole("summarizing", profile.Summarizing)
		printRole("general_big", profile.GeneralBig)
		printRole("general_mid", profile.GeneralMid)
		printRole("general_small", profile.GeneralSmall)
		return nil
	},
}

func init() {
	llmCmd.AddCommand(llmProfileShowCmd)
}
