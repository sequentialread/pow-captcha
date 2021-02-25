package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

var httpClient *http.Client
var captchaAPIURL *url.URL
var captchaChallenges []string

var items []string

// 5 bits of difficulty, 1 in 2^6 (1 in 32) tries will succeed on average.
//
// 8 bits of difficulty would be ok for apps that are never used on mobile phones, 6 is better suited for mobile apps
//
const captchaDifficultyLevel = 5

func main() {

	httpClient = &http.Client{
		Timeout: time.Second * time.Duration(5),
	}

	var err error
	captchaAPIURL, err = url.Parse("http://localhost:2370")
	if err != nil {
		panic(errors.New("can't start the app because can't parse captchaAPIURL"))
	}

	err = loadCaptchaChallenges()
	if err != nil {
		panic(errors.Wrap(err, "can't start the app because could not loadCaptchaChallenges():"))
	}

	_, err = ioutil.ReadFile("index.html")
	if err != nil {
		panic(errors.Wrap(err, "can't start the app because can't open the template file. Are you in the right directory? "))
	}

	http.HandleFunc("/", func(responseWriter http.ResponseWriter, request *http.Request) {

		// The user submitted a POST request, attempting to add a new item to the list
		if request.Method == "POST" {

			// Ask the captcha server if the user's proof of work result is legit,
			// and if not, return HTTP 400 Bad Request
			err := request.ParseForm()
			if err == nil {
				err = validateCaptcha(request.Form.Get("challenge"), request.Form.Get("nonce"))
			}

			if err != nil {
				responseWriter.WriteHeader(400)
				responseWriter.Write([]byte(fmt.Sprintf("400 bad request: %s", err)))
				return
			}

			// Validation passed, add the user's new item to the list
			items = append(items, request.Form.Get("item"))
		}

		// if it looks like we will run out of challenges soon, then kick off a goroutine to go get more in the background
		// note that in a real application in production, you would want to use a lock or mutex to ensure that
		// this only happens once if lots of requests come in at the same time
		if len(captchaChallenges) > 0 && len(captchaChallenges) < 5 {
			go loadCaptchaChallenges()
		}

		// if we somehow completely ran out of challenges, load more synchronously
		if captchaChallenges == nil || len(captchaChallenges) == 0 {
			err = loadCaptchaChallenges()
			if err != nil {
				log.Printf("loading captcha challenges failed: %v", err)
				responseWriter.WriteHeader(500)
				responseWriter.Write([]byte("captcha api error"))
				return
			}
		}

		// This gets & consumes the next challenge from the begining of the slice
		challenge := captchaChallenges[0]
		captchaChallenges = captchaChallenges[1:]

		// render the page HTML & output the result to the web browser
		htmlBytes, err := renderPageTemplate(challenge)
		if err != nil {
			log.Printf("renderPageTemplate(): %v", err)
			responseWriter.WriteHeader(500)
			responseWriter.Write([]byte("500 internal server error"))
			return
		}
		responseWriter.Write(htmlBytes)
	})

	log.Println("📋  Todo List example application listening on port 8080")

	err = http.ListenAndServe(":8080", nil)

	// if got this far it means server crashed!
	panic(err)
}

func renderPageTemplate(challenge string) ([]byte, error) {

	// in a real application in production you would read the template file & parse it 1 time when the app starts
	// I'm doing it for each request here just to make it easier to hack on it while its running 😇
	indexHTMLTemplateString, err := ioutil.ReadFile("index.html")
	if err != nil {
		return nil, errors.Wrap(err, "can't open the template file. Are you in the right directory? ")
	}
	pageTemplate, err := template.New("master").Parse(string(indexHTMLTemplateString))
	if err != nil {
		return nil, errors.Wrap(err, "can't parse the template file: ")
	}

	// constructing an instance of an anonymous struct type to contain all the data
	// that we need to pass to the template
	pageData := struct {
		Challenge  string
		Items      []string
		CaptchaURL string
	}{
		Challenge:  challenge,
		Items:      items,
		CaptchaURL: captchaAPIURL.String(),
	}
	var outputBuffer bytes.Buffer
	err = pageTemplate.Execute(&outputBuffer, pageData)
	if err != nil {
		return nil, errors.Wrap(err, "rendering page template failed: ")
	}

	return outputBuffer.Bytes(), nil
}

func loadCaptchaChallenges() error {

	query := url.Values{}
	query.Add("difficultyLevel", strconv.Itoa(captchaDifficultyLevel))

	loadURL := url.URL{
		Scheme:   captchaAPIURL.Scheme,
		Host:     captchaAPIURL.Host,
		Path:     filepath.Join(captchaAPIURL.Path, "GetChallenges"),
		RawQuery: query.Encode(),
	}

	captchaRequest, err := http.NewRequest("POST", loadURL.String(), nil)
	if err != nil {
		return err
	}

	response, err := httpClient.Do(captchaRequest)
	if err != nil {
		return err
	}

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode != 200 {
		return fmt.Errorf(
			"load proof of work captcha challenges api returned http %d: %s",
			response.StatusCode, string(responseBytes),
		)
	}

	err = json.Unmarshal(responseBytes, &captchaChallenges)
	if err != nil {
		return err
	}

	if len(captchaChallenges) == 0 {
		return errors.New("proof of work captcha challenges api returned empty array")
	}

	return nil
}

func validateCaptcha(challenge, nonce string) error {
	query := url.Values{}
	query.Add("challenge", challenge)
	query.Add("nonce", nonce)

	verifyURL := url.URL{
		Scheme:   captchaAPIURL.Scheme,
		Host:     captchaAPIURL.Host,
		Path:     filepath.Join(captchaAPIURL.Path, "Verify"),
		RawQuery: query.Encode(),
	}

	captchaRequest, err := http.NewRequest("POST", verifyURL.String(), nil)
	if err != nil {
		return err
	}

	response, err := httpClient.Do(captchaRequest)
	if err != nil {
		return err
	}

	if response.StatusCode != 200 {
		return errors.New("proof of work captcha validation failed")
	}
	return nil
}
