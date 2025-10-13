package config

import (
	"strconv"
	"strings"
)

type NetAddress struct {
	Host string
	Port int
}

type Config struct {
	Address         NetAddress
	RedirectAddress NetAddress
}

func (n NetAddress) String() string {
	return n.Host + ":" + strconv.Itoa(n.Port)
}

func (n *NetAddress) Set(flagValue string) error {
	res := strings.Split(flagValue, ":")
	n.Host = res[0]
	port, err := strconv.Atoi(res[1])
	n.Port = port

	return err
}

func GetConfig(c *Config) *Config {
	cfg := &Config{
		Address:         c.Address,
		RedirectAddress: c.RedirectAddress,
	}
	return cfg
}
