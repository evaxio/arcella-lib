package otel

type ConfigOtello struct {
	Url               string            `yaml:"url"            env:"OTELLO_URL"`
	ServiceName       string            `yaml:"serviceName"    env:"OTELLO_SERVICE_NAME"`
	UnqServiceName    string            `yaml:"unqServiceName" env:"OTELLO_UNQ_SERVICE_NAME"`
	Labels            map[string]string `yaml:"labels"         env:"OTELLO_LABELS"`
	Debug             bool              `yaml:"debug"          env:"OTELLO_DEBUG"` // Deprecated: reserved, not used.
	WithoutPrometheus bool
	SampleRatio       float64 `yaml:"sampleRatio" env:"OTELLO_SAMPLE_RATIO" default:"1.0"`
	TLS               bool    `yaml:"tls"         env:"OTELLO_TLS"`
}
