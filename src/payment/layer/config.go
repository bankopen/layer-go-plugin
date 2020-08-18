package layer

import (
	"encoding/json"
	"fmt"
	"os"
)

//Config structure
type Config struct {
	AccessKey               string
	SecretKey               string
	Environment             string
	SampleDataName          string
	SampleDataEmail         string
	SampleDataAmount        float32
	SampleDataCurrency      string
	SampleDataContactNumber string
}

//LoadConfiguration - load the configuration file
func LoadConfiguration(file string) Config {
	var config Config
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
}
