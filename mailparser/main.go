package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"path/filepath"
	"regexp"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Dict map[string]string

type Json map[string]interface{}

func main() {

	Configure()

	http.HandleFunc("/", Handler)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("OK")) })
	log.Println("Server started at port " + viper.GetString("port"))
	log.Fatal(http.ListenAndServe(":"+viper.GetString("port"), nil))
}

func Configure() {
	viper.SetDefault("logging", map[string]string{"path": ".", "type": "none"})
	viper.SetDefault("port", "8081")

	flag.Bool("v", false, "Verbose")

	viper.RegisterAlias("verbose", "v")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/mailparser/")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Fatalf("%v\n", err)
	}

	if viper.GetBool("verbose") {
		log.Println("Verbose: ON")
		log.Printf("Config: `%s`\n", viper.ConfigFileUsed())
	} else {
		log.Println("Verbose: OFF")

	}
}

func Handler(w http.ResponseWriter, r *http.Request) {
	dump, _ := httputil.DumpRequest(r, true)
	body, _ := ioutil.ReadAll(r.Body)
	hash := md5.Sum(dump)

	log.Printf("Received request - %x\n", hash)

	data, err := ParseMessage(body)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(fmt.Sprintf("Error: %v", err)))
		if viper.GetBool("verbose") {
			log.Printf("Parsing failed: %s\n", err)
		}
		fn := filepath.Join(viper.GetString("logging.path"), fmt.Sprintf("%x.dump", hash))
		err := ioutil.WriteFile(fn, dump, 0644)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Saved request: %s", fn)
		return
	}
	data["account"] = r.RequestURI[1:]

	if viper.GetBool("verbose") {
		log.Printf("Parsed - %v\n", data)
	}

	jb, _ := json.Marshal(data)

	if viper.GetString("webhook") != "" {
		resp, post_err := http.Post(viper.GetString("webhook"), "application/json", bytes.NewReader(jb))

		if post_err != nil {
			log.Printf("Webhook failed: %s (%s)", viper.GetString("webhook"), post_err)
			w.WriteHeader(503)
			w.Write([]byte("Error"))
			return
		}

		if viper.GetBool("verbose") {
			rb, _ := io.ReadAll(resp.Body)
			log.Printf("Webhook response: %s", string(rb))
		}

	}
	w.Write([]byte("OK"))
	if viper.GetBool("verbose") {
		log.Println("Successfully processed")
	}
	if viper.GetString("logging.type") == "all" {
		fn := filepath.Join(viper.GetString("logging.path"), fmt.Sprintf("%x.dump", hash))
		err := ioutil.WriteFile(fn, dump, 0644)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Saved request %s", fn)
	}
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

func ParseMessage(body []byte) (data Dict, err error) {

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

	for _, p := range viper.GetStringSlice("patterns") {
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
		if viper.GetBool("verbose") {
			log.Printf("--- NO MATCHES ---\n%s\n---\n", wording)
		}
		err = errors.New("no matches found")
		if viper.GetString("logging") == "failure" {
			hash := json_["envelope"].(map[string]interface{})["md5"].(string)
			err := ioutil.WriteFile(hash+".json", body, 0644)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Saved failed request %s.json", hash)
		}
		return
	}

	return
}
