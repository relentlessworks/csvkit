package config

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
	Addr   string
	NoAuth bool
}

func Load() Config {
	var c Config
	flag.StringVar(&c.Addr, "addr", getEnv("CSVKIT_ADDR", ":8080"), "listen address")
	flag.BoolVar(&c.NoAuth, "no-auth", getEnvBool("CSVKIT_NO_AUTH", false), "disable auth (dev mode)")
	flag.Parse()
	return c
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		switch v {
		case "1", "true", "TRUE", "True":
			return true
		case "0", "false", "FALSE", "False":
			return false
		}
	}
	return fallback
}

func (c Config) String() string {
	return fmt.Sprintf("addr=%s no-auth=%v", c.Addr, c.NoAuth)
}
