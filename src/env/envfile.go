package env

import (
	"log"

	"github.com/spf13/viper"
)

type ENV struct {
	SUPABASE_DB_URL 			string `mapstructure:"SUPABASE_DB_URL"`
	SUPABASE_PROJECT_URL		string `mapstructure:"SUPABASE_DB_URL"`
	SUPABASE_PUBLISHABLE_KEY	string `mapstructure:"SUPABASE_DB_URL"`
	SUPABASE_ANON_KEY			string `mapstructure:"SUPABASE_DB_URL"`
	JWT_KEY						string `mapstructire:"JWT_KEY"`
}

func NewEnv() *ENV{
	env := ENV{}
	viper.SetConfigFile(".env")

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("Can't find the file .env : ", err)
	}

	err = viper.Unmarshal(&env)
	if err != nil {
		log.Fatal("Environment can't be loaded: ", err)
	}

	return &env
}