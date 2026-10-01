package branding

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Organization struct {
	LegalName   string `yaml:"legal_name" json:"legal_name"`
	TradingName string `yaml:"trading_name" json:"trading_name"`
	Tagline     string `yaml:"tagline" json:"tagline"`
	BaseDomain  string `yaml:"base_domain" json:"base_domain"`
}

type Branding struct {
	PrimaryColor   string `yaml:"primary_color" json:"primary_color"`
	SecondaryColor string `yaml:"secondary_color" json:"secondary_color"`
	AccentColor    string `yaml:"accent_color" json:"accent_color"`
	BackgroundColor string `yaml:"background_color" json:"background_color"`
	CardColor      string `yaml:"card_color" json:"card_color"`
	TextColor      string `yaml:"text_color" json:"text_color"`
	LogoIcon       string `yaml:"logo_icon" json:"logo_icon"`
	FontFamily     string `yaml:"font_family" json:"font_family"`
}

type Modules struct {
	EnableCoreInvest      bool `yaml:"enable_core_invest" json:"enable_core_invest"`
	EnableFelineWorkforce bool `yaml:"enable_feline_workforce" json:"enable_feline_workforce"`
	EnablePebbleGateway   bool `yaml:"enable_pebble_gateway" json:"enable_pebble_gateway"`
}

type Config struct {
	Organization Organization `yaml:"organization" json:"organization"`
	Branding     Branding     `yaml:"branding" json:"branding"`
	Modules      Modules      `yaml:"modules" json:"modules"`
}

func DefaultConfig() *Config {
	return &Config{
		Organization: Organization{
			LegalName:   "Orange Cat Investments LLC",
			TradingName: "Orange Cat Investments",
			Tagline:     "SMB Self-Hosted Platform",
			BaseDomain:  "oci.local",
		},
		Branding: Branding{
			PrimaryColor:    "#f97316",
			SecondaryColor:  "#ea580c",
			AccentColor:     "#3b82f6",
			BackgroundColor: "#0f172a",
			CardColor:       "#1e293b",
			TextColor:       "#f8fafc",
			LogoIcon:        "🐈",
			FontFamily:      "Inter, system-ui, sans-serif",
		},
		Modules: Modules{
			EnableCoreInvest:      true,
			EnableFelineWorkforce: true,
			EnablePebbleGateway:   true,
		},
	}
}

func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Fallback to default
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
