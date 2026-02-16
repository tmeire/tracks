package mail

type Config struct {
	DeliveryMethod string            `json:"delivery_method"`
	SMTP           SMTPConfig        `json:"smtp_settings"`
	Defaults       DefaultsConfig    `json:"defaults"`
}

type SMTPConfig struct {
	Address            string `json:"address"`
	Port               int    `json:"port"`
	UserName           string `json:"user_name"`
	Password           string `json:"password"`
	Authentication     string `json:"authentication"`
	EnableStartTLSAuto bool   `json:"enable_starttls_auto"`
}

type DefaultsConfig struct {
	From string `json:"from"`
}
