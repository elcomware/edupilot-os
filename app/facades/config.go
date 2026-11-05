package facades

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/goravel/framework/contracts/config"
	"github.com/goravel/framework/support"
	"github.com/spf13/viper"
)

// The v1.18 configuration provider exits with status 0 on malformed .env files
// or invalid APP_KEY values. Validate the identical inputs before its first use
// so deployment commands cannot terminate successfully before reaching main.
var checkEnvironment = sync.OnceFunc(func() {
	values := viper.New()
	values.AutomaticEnv()
	path := support.EnvFilePath
	if _, err := os.Stat(path); err == nil {
		values.SetConfigType("env")
		values.SetConfigFile(path)
		if err := values.ReadInConfig(); err != nil {
			panic(fmt.Sprintf("could not parse configuration file %s", path))
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		panic(fmt.Sprintf("could not read configuration file %s", path))
	}
	if !support.DontVerifyAppKey {
		key, ok := values.Get("APP_KEY").(string)
		if !ok || len(key) != 32 {
			panic("APP_KEY must contain 32 bytes; initialize it with ./artisan key:generate")
		}
	}
})

func Config() config.Config {
	checkEnvironment()
	return App().MakeConfig()
}
