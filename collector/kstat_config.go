package collector

import (
	"gopkg.in/yaml.v2"
	"os"
	"regexp"
)

type kstatConfig struct {
	KstatModules []KstatModule `yaml:"kstat_modules"`
}

type KstatModule struct {
	ID         string      `yaml:"id"`
	KstatNames []KstatName `yaml:"kstat_names"`
}

type KstatName struct {
	ID          re          `yaml:"id"`
	LabelString string      `yaml:"label_string"`
	KstatStats  []KstatStat `yaml:"kstat_stats"`
}

type KstatStat struct {
	ID          string  `yaml:"id"`
	Help        string  `yaml:"help"`
	Suffix      string  `yaml:"suffix"`
	ScaleFactor float64 `yaml:"scale_factor"`
}

func (cfg *kstatConfig) init() error {
	var (
		cfgFile kstatConfig
	)

	yamlFile, err := os.ReadFile(kstatCfgFilePath())

	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, &cfgFile)
	if err != nil {
		return err
	}

	for _, cfgModule := range cfgFile.KstatModules {
		m := KstatModule{}
		m.ID = cfgModule.ID
		for _, cfgName := range cfgModule.KstatNames {
			n := KstatName{}
			n.ID = cfgName.ID
			if len(cfgName.LabelString) == 0 {
				n.LabelString = "instance"
			} else {
				n.LabelString = cfgName.LabelString
			}
			for _, cfgStat := range cfgName.KstatStats {
				s := KstatStat{}

				s.ID = cfgStat.ID

				if len(cfgStat.Help) == 0 {
					s.Help = cfgModule.ID + "::" + cfgName.ID.String() + ":" + cfgStat.ID
				} else {
					s.Help = cfgStat.Help
				}
				if cfgStat.ScaleFactor == 0 {
					s.ScaleFactor = 1
				} else {
					s.ScaleFactor = cfgStat.ScaleFactor
				}
				if len(cfgStat.Suffix) == 0 {
					s.Suffix = "total"
				} else {
					s.Suffix = cfgStat.Suffix
				}
				n.KstatStats = append(n.KstatStats, s)
			}
			m.KstatNames = append(m.KstatNames, n)
		}
		cfg.KstatModules = append(cfg.KstatModules, m)
	}
	return nil
}

type re struct {
	*regexp.Regexp
}

func (r *re) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	regex, err := regexp.Compile(s)
	if err != nil {
		return err
	}
	r.Regexp = regex
	return nil
}
