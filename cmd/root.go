package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

const banner = `
   ▄████████    ▄████████ ███▄▄▄▄       ███      ▄█  ███▄▄▄▄      ▄████████  ▄█
  ███    ███   ███    ███ ███▀▀▀██▄ ▀█████████▄ ███  ███▀▀▀██▄   ███    ███ ███
  ███    █▀    ███    █▀  ███   ███    ▀███▀▀██ ███▌ ███   ███   ███    █▀  ███
  ███         ▄███▄▄▄     ███   ███     ███   ▀ ███▌ ███   ███  ▄███▄▄▄     ███
▀███████████ ▀▀███▀▀▀     ███   ███     ███     ███▌ ███   ███ ▀▀███▀▀▀     ███
         ███   ███    █▄  ███   ███     ███     ███  ███   ███   ███    █▄  ███
   ▄█    ███   ███    ███ ███   ███     ███     ███  ███   ███   ███    ███ ███▌    ▄
 ▄████████▀    ██████████  ▀█   █▀     ▄████▀   █▀    ▀█   █▀    ██████████ █████▄▄██
                                                                         ▀
        The quality safety net for AI-assisted codebases, not a gatekeeper
`

var rootCmd = &cobra.Command{
	Use:   "sentinel",
	Short: "Sentinel - A risk analysis tool for AI-assisted codebases.",
	Long: banner + `
Sentinel uses a hybrid approach (Heuristics + Machine Learning) to
detect elevated-risk patterns in source code.

` + "── LANGUAGES SUPPORTED ──────────────────────────────────────────" + `
  Python, Java, JavaScript, TypeScript, Go, Rust, C++, Ruby, PHP,
  C#, Kotlin, Swift

` + "── QUICK START ──────────────────────────────────────────────────" + `
  # Scan current directory with default settings
  $ ./sentinel scan

  # Scan with high strictness
  $ ./sentinel scan --path ./src --threshold 0.85 --verbose

  # CI/CD: Scan only changed files and fail on detection
  $ ./sentinel scan --git-diff main --fail-on-detection

` + "──────────────────────────────────────────────────────────────────",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sentinel.yaml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose output with detailed logging")
	rootCmd.PersistentFlags().Bool("json", false, "output results in JSON format for CI/CD integration")

	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding verbose flag: %v\n", err)
	}
	if err := viper.BindPFlag("json", rootCmd.PersistentFlags().Lookup("json")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding json flag: %v\n", err)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".sentinel")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil && viper.GetBool("verbose") {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
