package smsc

type ConfigSMSC struct {
	SMSC          string `yaml:"server"        env:"SMSC_SERVER"` // "localhost:2775"
	SystemID      string `yaml:"login"         env:"SMSC_LOGIN"`
	Password      string `yaml:"password"      env:"SMSC_PASS"`
	SystemType    string `yaml:"type"          env:"SMSC_TYPE"           default:"cp"`
	SourceAddr    string `yaml:"sourceAddr"    env:"SMSC_SOURCE_ADDR"    default:"79232501250"`
	ResultTimeout string `yaml:"resultTimeout" env:"SMSC_RESULT_TIMEOUT" default:"5s"`
	DryRun        bool   `yaml:"dryRun"`
}
