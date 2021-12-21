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

type Dict map[string]string

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

	body, _ := ioutil.ReadAll(r.Body)

	data, err := parseMessage(body)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(fmt.Sprintf("Invalid payload - %v", err)))
		if *verbose {
			log.Printf("Reponded with HTTP/400 - Invalid payload - %s\n", err)
			//log.Printf("Payload:\n%s\n", string(body))
		}
		return
	}
	data["account"] = r.RequestURI[1:]

	if *verbose {
		log.Printf("Parsed - %v\n", data)
	}

	jb, _ := json.Marshal(data)
	resp, post_err := http.Post(config.Webhook, "application/json", bytes.NewReader(jb))

	if post_err != nil {
		log.Printf("Failed to POST to %s (%s)", config.Webhook, post_err)
		w.WriteHeader(503)
		w.Write([]byte("Error"))
		return
	}

	if *verbose {
		rb, _ := io.ReadAll(resp.Body)
		log.Printf("POST response: %s", string(rb))
	}

	if *verbose {
		log.Println("Reponded with HTTP/200 - OK")
	}
	w.Write([]byte("OK"))
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

func parseMessage(body []byte) (data Dict, err error) {

	if data == nil {
		data = make(Dict)
	}
	decoder := json.NewDecoder(bytes.NewBuffer(body))
	decoder.UseNumber()

	json_ := make(map[string]interface{})
	err = decoder.Decode(&json_)
	if err != nil {
		return
	}

	if _, ok := json_["html"]; !ok {
		err = errors.New("missing html")
		return
	}

	wording := stripHtml(json_["html"].(string))

	if _, ok := json_["headers"]; !ok {
		err = errors.New("missing headers")
		return
	}

	dt, _ := time.Parse(time.RFC1123Z, json_["headers"].(map[string]interface{})["date"].(string))
	data["created"] = dt.Format("2006-01-02T15:04:05Z07:00")

	for _, p := range config.Patterns {
		r := regexp.MustCompile(p)
		n := r.SubexpNames()
		s := ""
		for i, m := range r.FindStringSubmatch(wording) {
			switch n[i] {
			case "":
				continue
			case "negative":
				s = "-"
				continue
			case "amount":
				data[n[i]] = s + m
				continue
			}
			data[n[i]] = m
		}
	}

	if _, k := data["amount"]; !k {
		if *verbose {
			log.Printf("--- NO MATCHES ---\n%s\n---\n", wording)
		}
		err = errors.New("no matches found")
		return
	}

	return
}
