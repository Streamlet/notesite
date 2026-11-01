package config

import (
	"fmt"
	"regexp"

	"github.com/BurntSushi/toml"
	"github.com/Streamlet/NoteIsSite/util"
)

type SiteConfig struct {
	Server   ServerConfig   `toml:"server"`
	Template TemplateConfig `toml:"template"`
	Note     NoteConfig     `toml:"note"`
}

type ServerConfig struct {
	Port uint   `toml:"port"` // If port is specified, sock MUST be empty string.
	Sock string `toml:"sock"` // sIf sock is specified, port MUST be 0.
}

type TemplateConfig struct {
	TemplateRoot     string   `toml:"template_root"`
	StaticDirs       []string `toml:"static_dirs"`
	IndexTemplate    string   `toml:"index_template"`
	CategoryTemplate string   `toml:"category_template"`
	ContentTemplate  string   `toml:"content_template"`
	ErrorPage404     string   `toml:"404"` // optional
	ErrorPage500     string   `toml:"500"` // optional
}

type NoteConfig struct {
	NoteRoot           string `toml:"note_root"`
	CategoryConfigFile string `toml:"category_config_file"`
	ResourceConfigFile string `toml:"resource_config_file"`
	NoteFilePattern    string `toml:"note_file_pattern"`
	NoteFileRegExp     *regexp.Regexp
}

var siteConfig *SiteConfig

func LoadSiteConfig(configPath string) error {
	util.Assert(siteConfig == nil, "duplicate loading site config")
	conf := new(SiteConfig)
	if _, err := toml.DecodeFile(configPath, conf); err != nil {
		return err
	}
	if conf.Server.Port == 0 && conf.Server.Sock == "" {
		return fmt.Errorf("server.port or sock MUST be set")
	}
	if conf.Server.Port > 0 && conf.Server.Sock != "" {
		return fmt.Errorf("server.port and server.sock can NOT be both set")
	}
	if conf.Template.TemplateRoot == "" {
		return fmt.Errorf("template.template_root MUST be set")
	}
	if conf.Template.IndexTemplate == "" {
		return fmt.Errorf("template.index_template MUST be set")
	}
	if conf.Template.CategoryTemplate == "" {
		return fmt.Errorf("template.category_template MUST be set")
	}
	if conf.Template.ContentTemplate == "" {
		return fmt.Errorf("template.content_template MUST be set")
	}
	if conf.Note.NoteRoot == "" {
		return fmt.Errorf("note.note_root MUST be set")
	}
	if conf.Note.CategoryConfigFile == "" {
		return fmt.Errorf("note.category_config_file MUST be set")
	}
	if conf.Note.ResourceConfigFile == "" {
		return fmt.Errorf("note.resource_config_file MUST be set")
	}
	if conf.Note.NoteFilePattern == "" {
		return fmt.Errorf("note.note_file_pattern MUST be set")
	}
	regex, err := regexp.Compile(conf.Note.NoteFilePattern)
	if err != nil {
		return err
	}
	conf.Note.NoteFileRegExp = regex
	siteConfig = conf
	return nil
}

func GetSiteConfig() *SiteConfig {
	util.Assert(siteConfig != nil, "use site config before loading")
	return siteConfig
}
