package sender

type ConfigMailSender struct {
	Server   string `yaml:"server"  env:"MAIL_SERVER"` //
	User     string `yaml:"login"   env:"MAIL_LOGIN"`  //
	Password string `yaml:"pass"    env:"MAIL_PASS"`   //
	WithSSL  bool   `yaml:"ssl"     env:"MAIL_SSL"`    //
	Timeout  string `yaml:"timeout" env:"MAIL_TIMEOUT" default:"30s"`

	DryRun bool `yaml:"dryRun"`
}
