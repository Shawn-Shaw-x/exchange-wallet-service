package config

func NewTestDbConfig() *DBConfig {
	return &DBConfig{
		Host:     "106.15.105.133",
		Port:     5432,
		Name:     "testdb",
		User:     "ray",
		Password: "Feilin0)",
	}
}
