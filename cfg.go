package main

import (
	"errors"
	"flag"
	"os"
)

type Config struct {
	URL      string
	User     string
	Password string
}

func parse(config *Config) error {
	flag.StringVar(&config.URL, "url", "192.168.178.1", "FritzBox URL")
	flag.StringVar(&config.User, "user", os.Getenv("FB_USERNAME"), "user name")
	flag.StringVar(&config.Password, "password", os.Getenv("FB_PASSWORD"), "password")
	flag.Parse()

	if len(config.User) == 0 || len(config.Password) == 0 {
		return errors.New("please enter user name / password")
	}

	return nil
}
