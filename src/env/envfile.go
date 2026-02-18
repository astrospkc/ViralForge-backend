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
	JWT_KEY						string `mapstructure:"JWT_KEY"`
	S3_BUCKET_NAME				string `mapstructure:"S3_BUCKET_NAME"`
	AWS_SECRET_ACCESS_KEY		string `mapstructure:"S3_BUCKET_NAME"`
	AWS_ACCESS_KEY_ID			string `mapstructure:"S3_BUCKET_NAME"`
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