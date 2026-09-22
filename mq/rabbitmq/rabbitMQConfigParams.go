package rabbitmq

type ConfigRabbitMQ struct {
	Url                string `yaml:"url"                 env:"RMQ_URL"`
	Consumer           string `yaml:"consumer"            env:"RMQ_CONSUMER"`
	Queue              string `yaml:"queue"               env:"RMQ_QUEUE"`
	Exchange           string `yaml:"exchange"            env:"RMQ_EXCHANGE"`
	ConcurrencyCount   int    `yaml:"concurrencyCount"    env:"RMQ_CONCURRENCY_COUNT"  default:"1" `
	RetryDelay         string `yaml:"retry"               env:"RMQ_RETRY_DELAY"        default:"5s"`
	ConfirmTimeout     string `yaml:"confirmTimeout"      env:"RMQ_CONFIRM_TIMEOUT"    default:"10s"`
	DeadLetterExchange string `yaml:"deadLetterExchange"  env:"RMQ_DEAD_LETTER_EXCHANGE"`
	Quorum             bool   `yaml:"quorum"              env:"RMQ_QUORUM"             default:"true"`
	Disabled           bool   `yaml:"disabled,omitempty"`
	Debug              bool   `yaml:"debug,omitempty"`
}
