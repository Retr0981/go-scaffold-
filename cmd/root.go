package cmd

import (
	"context"
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
	version = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "goscaffold",
	Short: "Advanced Go project scaffolding with AI integration",
	Long: `goscaffold is a cross-platform CLI tool that creates project structures 
and imports AI-generated code with validation, backups, and git integration.

Supports multiple input formats, interactive TUI, and plugin-based validators.`,
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogging()
		initConfig()
	},
}

func Execute() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info("Shutting down gracefully...")
		cancel()
		time.Sleep(2 * time.Second)
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
	log.SetTimeFormat(time.Kitchen)
	log.SetPrefix("goscaffold")

	// Colored output based on terminal
	if !isTerminal() {
		log.SetFormatter(log.TextFormatter)
	}
}

func initConfig() {
	viper.SetEnvPrefix("GOSCAFFOLD")
	viper.AutomaticEnv()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".goscaffold")
	}

	// Set defaults
	viper.SetDefault("backup.enabled", true)
	viper.SetDefault("backup.retention", "7d")
	viper.SetDefault("git.auto_commit", false)
	viper.SetDefault("git.default_branch", "main")
	viper.SetDefault("watch.interval", "5s")
	viper.SetDefault("ui.theme", "auto")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Warn("Error reading config", "error", err)
		}
	}

	log.Debug("Config initialized", "file", viper.ConfigFileUsed())
}

func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	return err == nil && (fileInfo.Mode()&os.ModeCharDevice) != 0
}
