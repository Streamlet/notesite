package config

import (
	"regexp"

	"github.com/BurntSushi/toml"
)

type CategoryConfig struct {
	DisplayName     string `toml:"display_name"`
	Name            string `toml:"name"`
	Index           string `toml:"index"`
	Descending      *bool  `toml:"descending"`
	NoteFilePattern string `toml:"note_file_pattern"`
	NoteFileRegExp  *regexp.Regexp
}

func GetCategoryConfig(dirPath string) (*CategoryConfig, error) {
	configPath := dirPath + "/" + GetSiteConfig().Note.CategoryConfigFile
	conf := new(CategoryConfig)
	if _, err := toml.DecodeFile(configPath, conf); err != nil {
		return nil, err
	}
	if conf.NoteFilePattern != "" {
		regex, err := regexp.Compile(conf.NoteFilePattern)
		if err != nil {
			return nil, err
		}
		conf.NoteFileRegExp = regex
	}
	return conf, nil
}
