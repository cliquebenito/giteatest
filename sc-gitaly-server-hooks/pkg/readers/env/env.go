package env

import (
	"fmt"
	"os"
	"strings"
)

type Reader struct {
	envs map[string]string
}

func NewEnvReader() Reader {
	reader := Reader{
		envs: make(map[string]string),
	}
	reader.Read()
	return reader
}

func (r Reader) Read() {
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		r.envs[pair[0]] = pair[1]
	}
}

func (r Reader) GetByKey(key string) (string, error) {
	value, ok := r.envs[key]
	if !ok {
		return "", fmt.Errorf("no env value found for key: %s", key)
	}
	return value, nil
}

func (r Reader) GetByKeyWithoutError(key string) string {
	return r.envs[key]
}
