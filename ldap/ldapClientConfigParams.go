package ldap

type ConfigLDAPClient struct {
	Url           string `yaml:"url"      env:"LDAP_URL"`
	User          string `yaml:"user"     env:"LDAP_USER"`
	Pass          string `yaml:"pass"     env:"LDAP_PASS"`
	Base          string `yaml:"base"     env:"LDAP_BASE"`
	Insecure      bool   `yaml:"insecure" env:"LDAP_INSECURE"`
	SearchTimeout string `yaml:"searchTimeout" env:"LDAP_SEARCH_TIMEOUT" default:"10s"`
}
