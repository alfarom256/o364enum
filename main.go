package main

import (
	"flag"
	"math/rand"
	"time"
	"log"
	"net/http"
	url2 "net/url"
	"io/ioutil"
	"strings"
	"fmt"
	"github.com/google/uuid"
	"crypto/tls"
	"encoding/json"
)

var (
	seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	proxy = false
)

const (
	JSON_TEXT = `{
"skuId": "3b555118-da6a-4418-894f-7df1e2096870",
"emailAddress": "%s",
"culture": "en-us",
"skipVerificationEmail": true
}`
	O365_HOST = "https://signup.microsoft.com/api/signupservice/usersignup?api-version=1&client-request-id=%s&culture=en-us"

	//16 Random mixed case letters and digits + ".0.4"
	MS_CV = "%s.0.4"

	charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	default_sleep = 1

)

type UserJSON struct {
	SKU_ID string `json:"skuId"`
	Tenant string `json:"tenantRegion"`
	TenantID int `json:"tenantId,omitempty"`
	EmailVerified bool `json:"isEmailVerifiedTenant"`
	SignupRequestToken string `json:"signupRequestToken"`
	HttpStatuscode int `json:"httpStatuscode"`
	ResponseCode string `json:"responseCode"`
	Message string `json:"message"`
	CorrelationId string `json:"correlationId"`
}

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func String(length int) string {
	return StringWithCharset(length, charset)
}

func main() {
	var proxy_host = flag.String("proxy", "", "proxy host or IP to use, i.e. http://127.0.0.1:9050")
	var insecure = flag.Bool("k", false, "allow insecure requests")
	var user_file = flag.String("userfile", "", "file containing users to try")
	var sleep_sec = flag.Int("sleep", default_sleep, "sleep time in seconds between requests (default: 1)")
	flag.Parse()

	if *user_file == "" {
		log.Fatal("Error, requires a user file")
	}
	if *sleep_sec == default_sleep {
		log.Println("Using default sleep value of 1")
	} else if *sleep_sec < 0{
		log.Fatal("Error, can't negative sleep!! :|")
	}


	client := http.Client{}

	if *proxy_host != ""{
		proxy = true
		log.Printf("Using proxy %s\n", *proxy_host)
		if url, err := url2.Parse(*proxy_host); err == nil {
			if *insecure {
				client.Transport = &http.Transport{
					Proxy: http.ProxyURL(url),
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
			} else {
				client.Transport = &http.Transport{
					Proxy: http.ProxyURL(url),
				}
			}
		} else {
			log.Fatal(err)
		}
	} else if *proxy_host == "" && (*insecure) {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}


	log.Println("Attempting to read users file")
	b_arr, err := ioutil.ReadFile(*user_file)
	if err != nil {
		log.Fatal(err)
	}
	unames := strings.Split(string(b_arr), "\r\n")
	log.Printf("Testing %d usernames", len(unames))

	for u := range unames {
		json_body := fmt.Sprintf(JSON_TEXT, unames[u])
		host := fmt.Sprintf(O365_HOST, uuid.New())
		ms_cv := fmt.Sprintf(MS_CV, String(16))
		req, err := http.NewRequest("POST", host, strings.NewReader(json_body))

		req.Header.Set("MS_VC", ms_cv)
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Content-Length", string(len(json_body)))


		if err != nil {
			log.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		data, err := ioutil.ReadAll(resp.Body)
		var user = &UserJSON{}
		err = json.Unmarshal(data, user)
		if err != nil {
			log.Println("Error in HTTP response, unexpected JSON")
			log.Println(err)
			log.Println(string(data))
		}
		if user.HttpStatuscode == 409 {
			println("valid:"+unames[u])
		} else {
			println("invalid:"+unames[u])
		}
		time.Sleep(time.Duration(*sleep_sec) * time.Second)
	}
}


