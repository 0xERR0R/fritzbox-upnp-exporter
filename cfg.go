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
	url := "192.168.178.1"
	if len(os.Getenv("FB_URL")) > 0 {
		url = os.Getenv("FB_URL")
	}
	flag.StringVar(&config.URL, "url", url, "FritzBox URL")
	flag.StringVar(&config.User, "user", os.Getenv("FB_USERNAME"), "user name")
	flag.StringVar(&config.Password, "password", os.Getenv("FB_PASSWORD"), "password")
	flag.Parse()

	if len(config.User) == 0 || len(config.Password) == 0 {
		return errors.New("please enter user name / password")
	}

	return nil
}
