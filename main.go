package main

import (
	"fmt"
	"net/http"
	"od2kproxy/od2kproxy"

	"github.com/spf13/viper"
)

func init() {
	viper.AddConfigPath(".")
	viper.SetConfigName("settings")
	viper.SetConfigType("json")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
}

func main() {
	portSetting := viper.GetString("http_port")
	if portSetting == "" {
		panic(fmt.Errorf("port is required"))
	}
	port := fmt.Sprintf(":%s", portSetting)

	client, err := od2kproxy.NewProxyClient()
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", client.Handler)
	err = http.ListenAndServe(port, nil)
	if err != nil {
		panic(err)
	}
}
