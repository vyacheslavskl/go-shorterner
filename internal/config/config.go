package config

import (
	"fmt"
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

func (n *NetAddress) String() string {
	return n.Host + ":" + strconv.Itoa(n.Port)
}

func (n *NetAddress) Set(flagValue string) error {
	res := strings.Split(flagValue, ":")
	fmt.Println(res)

	n.Host = res[0]
	port, err := strconv.Atoi(res[1])
	n.Port = port

	return err
}

func GetConfig() *Config {
	cfg := Config{
		Address:         NetAddress{Host: "localhost", Port: 8080},
		RedirectAddress: NetAddress{Host: "localhost", Port: 8081},
	}
	return &cfg
}
