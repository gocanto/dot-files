package release

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	NotesFile string
	Repo      string
	Tag       string
}

func LoadConfig(args []string) (Config, error) {
	args = trimArgumentSeparator(args)

	flags := pflag.NewFlagSet("release-macos-unsigned", pflag.ContinueOnError)
	flags.String("notes-file", "", "release notes file")
	flags.String("repo", "", "GitHub repository in owner/name format")
	flags.String("tag", "", "GitHub release tag")

	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}

	if flags.NArg() > 0 {
		return Config{}, fmt.Errorf("unknown argument: %s", flags.Arg(0))
	}

	v := viper.New()
	v.SetEnvPrefix("release")
	v.BindEnv("repo", "RELEASE_REPO")
	v.BindEnv("tag", "RELEASE_TAG")

	if err := v.BindPFlags(flags); err != nil {
		return Config{}, fmt.Errorf("bind flags: %w", err)
	}

	return Config{
		NotesFile: v.GetString("notes-file"),
		Repo:      v.GetString("repo"),
		Tag:       v.GetString("tag"),
	}, nil
}

func trimArgumentSeparator(args []string) []string {
	if len(args) > 0 && args[0] == "--" {
		return args[1:]
	}

	return args
}
