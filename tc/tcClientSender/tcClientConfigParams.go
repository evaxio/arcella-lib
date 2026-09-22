package trueConfClient

type ConfigTcClient struct {
	Server   string `yaml:"server"`
	User     string `yaml:"user"`
	Password string `yaml:"pass"`
	//
	ShowAnswers bool `yaml:"answers"`
	//
	InsecureSkipVerify bool `yaml:"insecureSkipVerify" default:"true"`
}
