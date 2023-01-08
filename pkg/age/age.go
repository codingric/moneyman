package age

import (
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"

	fage "filippo.io/age"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
)

var AgeKey *fage.X25519Identity

func Init(keypath string) {
	b, err := os.ReadFile(keypath) // just pass the file name
	if err != nil {
		log.Error().Err(err).Msgf("Failed to open age key: `%s`", keypath)
	}
	AgeKey, err = loadAgeKey(b)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to load key: `%s`", keypath)
	}
}

func loadAgeKey(b []byte) (a *fage.X25519Identity, e error) {
	// Remove comments
	re := regexp.MustCompile(`(s?)#.*\n`)
	c := re.ReplaceAll(b, nil)
	str := string(c) // convert content to a 'string'

	a, e = fage.ParseX25519Identity(strings.Trim(str, "\n"))
	return
}

func DecodeAge(s string, a *fage.X25519Identity) string {
	enc := strings.TrimPrefix(s, "age:")
	eb, _ := base64.StdEncoding.DecodeString(enc)
	r := bytes.NewReader(eb)
	d, _ := fage.Decrypt(r, a)
	b := &bytes.Buffer{}
	io.Copy(b, d)
	return b.String()
}

func AgeHookFunc(a *fage.X25519Identity) mapstructure.DecodeHookFuncType {
	// Wrapped in a function call to add optional input parameters (eg. separator)
	return func(
		f reflect.Type, // data type
		t reflect.Type, // target data type
		data interface{}, // raw data
	) (interface{}, error) {

		// Check if the data type matches the expected one
		if f.Kind() != reflect.String {
			return data, nil
		}

		// Check if the target type matches the expected one
		if t.Kind() != reflect.String {
			return data, nil
		}

		if !strings.HasPrefix(data.(string), "age:") {
			return data, nil
		}

		return DecodeAge(data.(string), a), nil
	}
}
