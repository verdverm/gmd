package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var llmProfilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "List configured LLM profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}
		if len(cfg.LLM.Profiles) == 0 {
			fmt.Println("No profiles configured")
			return nil
		}
		active := cfg.LLM.Profile
		if active == "" {
			active = "default"
		}
		for name, profile := range cfg.LLM.Profiles {
			marker := " "
			if name == active {
				marker = "*"
			}
			fmt.Printf("%s %s\n", marker, name)
			printRole("embedding", profile.Embedding)
			printRole("expansion", profile.Expansion)
			printRole("rerank", profile.Rerank)
			printRole("summarizing", profile.Summarizing)
			printRole("general_big", profile.GeneralBig)
			printRole("general_mid", profile.GeneralMid)
			printRole("general_small", profile.GeneralSmall)
		}
		return nil
	},
}

func init() {
	llmCmd.AddCommand(llmProfilesCmd)
}

func printRole(label string, rc *config.LLMRoleConfig) {
	if rc == nil || rc.Model == "" {
		return
	}
	fmt.Printf("    %-17s provider=%-15s model=%s\n", label, rc.Provider, rc.Model)
}
