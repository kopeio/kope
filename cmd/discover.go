package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover clusters",
	Long: `Discover cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("syntax: discover clusters")
	},
}

func init() {
	RootCmd.AddCommand(discoverCmd)
}
