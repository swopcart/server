package config

import "os"

type Env interface {
	Getenv(key string) (value string, ok bool)
}

type RealEnv struct{}

func (env *RealEnv) Getenv(key string) (value string, ok bool) {
	return os.LookupEnv(key)
}

type MockEnv map[string]string

func (env MockEnv) Getenv(key string) (value string, ok bool) {
	mockedValue, ok := env[key]
	if ok {
		return mockedValue, ok
	}

	return os.LookupEnv(key)
}
