package cmd

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	debug   bool
	trace   bool
	version = "1.0.0"
)

var rootCmd = &cobra.Command{
	Use:     "goscaffold",
	Short:   "Advanced Go project scaffolding with AI integration",
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogging()
		initConfig()
	},
}

func Execute() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info("Shutting down gracefully...")
		cancel()
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()

	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.goscaffold.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&trace, "trace", false, "enable trace logging")
}

func initLogging() {
	level := log.InfoLevel
	if debug {
		level = log.DebugLevel
	}
	if trace {
		level = log.TraceLevel
	}

	log.SetLevel(level)
	log.SetReportTimestamp(true)
	log.SetTimeFormat(time.RFC3339)
	log.SetPrefix("goscaffold")
}

func initConfig() {
	viper.SetEnvPrefix("GOSCAFFOLD")
	viper.AutomaticEnv()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
			viper.AddConfigPath(".")
			viper.SetConfigType("yaml")
			viper.SetConfigName(".goscaffold")
		}
	}

	// Defaults
	viper.SetDefault("backup.enabled", true)
	viper.SetDefault("backup.retention", "7d")
	viper.SetDefault("git.auto_commit", false)
	viper.SetDefault("git.default_branch", "main")
	viper.SetDefault("watch.interval", "5s")
	viper.SetDefault("ui.confirm_create", true)

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			log.Warn("Error reading config", "error", err)
		}
	}
}
