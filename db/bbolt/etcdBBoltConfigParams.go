package bbolt

type ConfigEtcdBBolt struct {
	FileName string `yaml:"file" env:"BBOLT_FILE" default:"bbolt.db"`
}
