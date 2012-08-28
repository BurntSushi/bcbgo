package bowdb

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	LibraryPath string
}

func openConfig(p string) (conf Config, err error) {
	f, err := os.Open(p)
	if err != nil {
		return conf,
			fmt.Errorf("Error opening the config file '%s': %s.", p, err)
	}

	decoder := json.NewDecoder(f)
	if err = decoder.Decode(&conf); err != nil {
		return conf,
			fmt.Errorf("Error decoding JSON in '%s': %s.", p, err)
	}
	if err = f.Close(); err != nil {
		return
	}
	return
}

func (conf Config) write(p string) (err error) {
	f, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("Error creating the config file '%s': %s.", p, err)
	}

	encoder := json.NewEncoder(f)
	if err = encoder.Encode(conf); err != nil {
		return
	}
	if err = f.Close(); err != nil {
		return
	}
	return
}
