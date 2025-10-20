package config

import (
	"strconv"
	"strings"
)

type NetAddress struct {
	Protocol string
	Host     string
	Port     int
}

type Config struct {
	Address         NetAddress
	RedirectAddress NetAddress
}

func (n NetAddress) String() string {
	return n.Protocol + n.Host + ":" + strconv.Itoa(n.Port)
}

func (n *NetAddress) Set(flagValue string) error {
	res := strings.Split(flagValue, ":")
	if len(res) == 3 {
		n.Protocol = res[0] + ":"
		n.Host = res[1]
		port, err := strconv.Atoi(res[2])
		if err != nil {
			return err
		}
		n.Port = port
	} else {
		n.Host = res[0]
		port, err := strconv.Atoi(res[1])
		if err != nil {
			return err
		}
		n.Port = port
		n.Port = port

	}
	return nil
}

func GetConfig(c *Config) *Config {
	cfg := &Config{
		Address:         c.Address,
		RedirectAddress: c.RedirectAddress,
	}
	return cfg
}
