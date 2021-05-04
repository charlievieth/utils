package main

import (
	goflag "flag"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	UnixSocketAddr string
	DatabaseDir    string
	LogDir         string
	LogLevel       zapcore.Level
	Log            *zap.Logger
}

func ConfigFromFlags() (*Config, error) {
	var conf Config
	set := pflag.NewFlagSet(filepath.Base(os.Args[0]), pflag.ExitOnError)

	set.StringVar(&conf.UnixSocketAddr, "addr", "", "unix socket address")
	set.StringVar(&conf.DatabaseDir, "db-dir", "", "database directory")

	lvlFlag := &goflag.Flag{
		Name:     "log-level",
		Usage:    "set the log level",
		Value:    &conf.LogLevel,
		DefValue: "INFO",
	}
	set.AddFlag(pflag.PFlagFromGoFlag(lvlFlag))

	// set.Var(&conf.LogLevel, "log-level", "set the log level")

	return nil, nil
}

func InitFlags() *pflag.FlagSet {

	set := pflag.NewFlagSet(filepath.Base(os.Args[0]), pflag.ExitOnError)

	// set.Var(value, name, usage)

	// set.AddGoFlag()
	// set.
	return set
}
