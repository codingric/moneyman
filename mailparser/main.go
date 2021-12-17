package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v2"
)

var (
	config_path = kingpin.Flag("config", "Config").Default("config.yaml").Short('c').String()
	verbose     = kingpin.Flag("verbose", "Verbosity").Short('v').Bool()
	config      ConfigType
)

type ConfigType struct {
	Port     string   `yaml:"port"`
	Webhook  string   `yaml:"webhook"`
	Patterns []string `yaml:"patterns"`
}

type Json map[string]interface{}

func main() {
	kingpin.Parse()

	config.Load(*config_path)

	http.HandleFunc("/", Handler)
	log.Println("Server started at port " + config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

func (c *ConfigType) Load(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	yaml.Unmarshal(data, c)
	if *verbose {
		log.Printf("Loaded %s\n", path)
	}
}

func Handler(w http.ResponseWriter, r *http.Request) {

	log.Println("Received request")

	data, err := parseMessage(r)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(fmt.Sprintf("Invalid payload - %v", err)))
		if *verbose {
			log.Printf("Reponded with HTTP/400 - Invalid payload - %s\n", err)
		}
		return
	}

	if *verbose {
		log.Printf("Parsed - %v\n", data)
	}

	if *verbose {
		log.Println("Reponded with HTTP/200 - OK")
	}
	w.Write([]byte("OK"))

	jb, _ := json.Marshal(data)
	resp, post_err := http.Post(config.Webhook, "application/json", bytes.NewReader(jb))

	if post_err != nil {
		log.Printf("Failed to POST to %s (%s)", config.Webhook, post_err)
		return
	}

	if *verbose {
		rb, _ := io.ReadAll(resp.Body)
		log.Printf("POST response: %s", string(rb))
	}
}

func parseJson(in io.Reader) Json {
	decoder := json.NewDecoder(in)
	decoder.UseNumber()
	j := make(map[string]interface{})
	for {
		err := decoder.Decode(&j)
		if err != nil {
			break
		}
	}
	return j
}

const remove = `<.*?>|\n+`
const spaces = `[ \t]+`

// This method uses a regular expresion to remove HTML tags.
func stripHtml(str string) string {
	r := regexp.MustCompile(remove)
	s := regexp.MustCompile(spaces)
	str = r.ReplaceAllString(str, "")
	return s.ReplaceAllString(str, " ")
}

func parseMessage(r *http.Request) (map[string]string, error) {

	j := parseJson(r.Body)

	if _, ok := j["html"]; !ok {
		return make(map[string]string), errors.New("missing html")
	}

	d := make(map[string]string)

	html := j["html"].(string)
	msg := stripHtml(html)

	d["account"] = r.RequestURI[1:]
	if _, ok := j["headers"]; !ok {
		return make(map[string]string), errors.New("missing headers")
	}

	dt, _ := time.Parse(time.RFC1123Z, j["headers"].(map[string]interface{})["date"].(string))
	d["created"] = dt.Format("2006-01-02T15:04:05Z07:00")

	for _, p := range config.Patterns {
		r := regexp.MustCompile(p)
		n := r.SubexpNames()
		s := ""
		for i, m := range r.FindStringSubmatch(msg) {
			switch n[i] {
			case "":
				continue
			case "negative":
				s = "-"
				continue
			case "amount":
				d[n[i]] = s + m
				continue
			}
			d[n[i]] = m
		}
	}

	if len(d) < 3 {
		return make(map[string]string), errors.New("no matches found")
	}

	return d, nil
}
