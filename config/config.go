package config

import (
	"bytes"

	"github.com/spf13/viper"
	"github.com/creasty/defaults"
)

var ConfigSample = `#Sample Opslevel Agent Config
integrationId: ""
payloadCheckId: ""
resync: 3600
deployments: true
statefulsets: false
daemonsets: false
jobs: false
cronjobs: false
services: false
ingress: false
configmaps: false
secrets: false
`

type Config struct {
	IntegrationId string `json:"integrationId"`
	PayloadCheckId string `json:"payloadCheckId"`
	Resync int `json:"resync"`
	Deployments bool `json:"deployments"`
	StatefulSets bool `json:"statefulsets"`
	DaemonSets bool `json:"daemonsets"`
	Jobs bool `json:"jobs"`
	CronJobs bool `json:"cronjobs"`
	Services bool `json:"services"`
	Ingress bool `json:"ingress"`
	Configmaps bool `json:"configmaps"`
	Secrets bool `json:"secrets"`
}

func New() (*Config, error) {
	c := &Config{}
	viper.Unmarshal(&c)
	if err := defaults.Set(c); err != nil {
		return c, err
	}
	return c, nil
}

func Default() (*Config, error) {
	c := &Config{}
	v := viper.New()
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer([]byte(ConfigSample)))
	v.Unmarshal(&c)
	if err := defaults.Set(c); err != nil {
		return c, err
	}
	return c, nil
}
