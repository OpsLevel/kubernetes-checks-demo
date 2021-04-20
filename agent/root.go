package agent

import (
	"os"
	"strings"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "opslevel-agent",
	Short: "Opslevel Example Kubernetes Agent",
	Long: `Opslevel Example Kubernetes Agent`,
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "./opslevel.yaml", "")
	rootCmd.PersistentFlags().String("logFormat", "JSON", "overrides environment variable 'OL_LOGFORMAT' (options [\"JSON\", \"TEXT\"])")
	rootCmd.PersistentFlags().String("logLevel", "INFO", "overrides environment variable 'OL_LOGLEVEL' (options [\"ERROR\", \"WARN\", \"INFO\", \"DEBUG\"])")
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	readConfig()
	setupLogging()
}

func readConfig() {
	if cfgFile != "" {
		if cfgFile == "." {
			viper.SetConfigType("yaml")
			viper.ReadConfig(os.Stdin)
			return
		} else {
			viper.SetConfigFile(cfgFile)
		}
	} else {
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		viper.SetConfigName("opslevel")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
	}

	viper.SetEnvPrefix("OL")
	viper.AutomaticEnv()
	viper.BindPFlags(rootCmd.Flags())

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}

func setupLogging() {
	logFormat := strings.ToLower(viper.GetString("logFormat"))
	logLevel := strings.ToLower(viper.GetString("logLevel"))

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if logFormat == "text" {
		output := zerolog.ConsoleWriter{Out: os.Stderr}
		log.Logger = log.Output(output)
	}

	switch {
		case logLevel == "error":
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		case logLevel == "warn":
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case logLevel == "debug":
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		default:
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}