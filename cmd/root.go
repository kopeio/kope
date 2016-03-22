package cmd

import (
	"fmt"
	"os"
	goflag "flag"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// This represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "kutil",
	Short: "Kubernetes toolbelt",
	Long: `Toolkit for Kubernetes`,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().AddGoFlagSet(goflag.CommandLine)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kutil.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".kutil")
	viper.AddConfigPath("$HOME")
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
