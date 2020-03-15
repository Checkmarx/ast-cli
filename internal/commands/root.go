package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	cfgFile string

	rootCmd = &cobra.Command{
		Use:   "cobra",
		Short: "A generator for Cobra based Applications",
		Long: `Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra.yaml)")
	rootCmd.PersistentFlags().StringP("author", "a", "YOUR NAME", "author name for copyright attribution")
	viper.BindPFlag("author", rootCmd.PersistentFlags().Lookup("author"))
	viper.SetDefault("author", "NAME HERE <EMAIL ADDRESS>")
}

//func errorAndExit(err error, msg string) {
//	log.WithFields(log.Fields{
//		"err": err,
//	}).Fatal(msg)
//}
