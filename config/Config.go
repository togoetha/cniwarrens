package config

import (
	"encoding/json"
	"fmt"
	"os"
)

var Cfg *Config

type Config struct {
	RemoteNodes map[string]NodeInfo
	NumServices int
	CNIPrefix   string
	Debug       bool
	TunnelPort  int
	TunDev      string
	UseXDP      bool
	UseWG       bool
}

type NodeInfo struct {
	PublicIPv6 string
	Port       int
	PublicMac  [6]uint8
}

func LoadConfig(filename string) error {
	fmt.Printf("Loading config %s\n", filename)
	file, err := os.Open(filename)
	if err != nil {
		//return err
	}
	decoder := json.NewDecoder(file)
	Cfg = &Config{}
	err = decoder.Decode(Cfg)
	if err != nil {
		fmt.Println(err.Error())
		//return err
	}

	return err
}
