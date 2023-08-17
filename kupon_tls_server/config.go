package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
)

type Config struct {
	Runtime string `yaml:"runtime"`
	NexusWS struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"nexusws"`
	TLSServer struct {
		ListenPort string `yaml:"listen_port"`
		ListenIP   string `yaml:"listen_ip"`
	} `yaml:"tls_server"`
}

func NewFromFile(file string, out interface{}) error {
	fileContents, err := ioutil.ReadFile(filepath.Clean(file))
	if err != nil {
		return err
	}

	return yaml.Unmarshal(fileContents, out)
}
