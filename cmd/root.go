package cmd

import (
	"errors"
	"fmt"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"path"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gumdrop",
	Short: "Create DigitalOcean droplets with custom cloud-init config",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := initConfig()
		if err != nil {
			log.Fatal(err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	//cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gumdrop.yaml)")
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initConfig() error {
	if cfgFile == "" {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			return err
		}
		cfgFile = path.Join(home, ".gumdrop.yaml")
	}
	viper.SetConfigFile(cfgFile)

	viper.SetEnvPrefix("gumdrop")
	viper.AutomaticEnv() // read in environment variables that match
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		//Check permissions:
		info, _ := os.Stat(viper.ConfigFileUsed())
		if info.Mode().Perm() != 0600 {
			log.Fatal("Insecure file permissions for config! Run: chmod 0600 ", viper.ConfigFileUsed())
		}
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		return errors.New("No config file found")
	}
	return nil
}
