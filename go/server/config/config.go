package config

type Configurations struct {
	Server   ServerConfigurations
	Database DatabaseConfigurations
	Api      ApiConfigurations
}

// ServerConfigurations exported
type ServerConfigurations struct {
	Port int
}

// DatabaseConfigurations exported
type DatabaseConfigurations struct {
	DBName     string
	DBUser     string
	DBPassword string
}

// ApiConfigurations exported
type ApiConfigurations struct {
	ServiceKey string
	EndPoint   string
	Operation  string
}
