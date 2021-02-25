package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"

	"golang.org/x/crypto/scrypt"
)

const batchSize = 1000
const deprecateAfterBatches = 10
const portNumber = 2370
const scryptCPUAndMemoryCost = 4096

// https://en.wikipedia.org/wiki/Scrypt
type ScryptParameters struct {
	CPUAndMemoryCost int `json:"N"`
	BlockSize        int `json:"r"`
	Paralellization  int `json:"p"`
	KeyLength        int `json:"klen"`
}

type Challenge struct {
	ScryptParameters
	Preimage        string `json:"i"`
	Difficulty      string `json:"d"`
	DifficultyLevel int    `json:"dl"`
}

var currentChallengesGeneration = 0
var challenges = map[string]int{}

func main() {

	scryptParameters := ScryptParameters{
		CPUAndMemoryCost: scryptCPUAndMemoryCost,
		BlockSize:        8,
		Paralellization:  1,
		KeyLength:        16,
	}

	http.HandleFunc("/GetChallenges", func(responseWriter http.ResponseWriter, request *http.Request) {

		if request.Method != "POST" {
			responseWriter.Header().Set("Allow", "POST")
			http.Error(responseWriter, "405 Method Not Allowed, try POST", http.StatusMethodNotAllowed)
		}

		currentChallengesGeneration++

		requestQuery := request.URL.Query()
		difficultyLevelString := requestQuery.Get("difficultyLevel")
		difficultyLevel, err := strconv.Atoi(difficultyLevelString)
		if err != nil {
			http.Error(
				responseWriter,
				fmt.Sprintf(
					"400 url param ?difficultyLevel=%s value could not be converted to an integer",
					difficultyLevelString,
				),
				http.StatusInternalServerError,
			)
			return
		}

		toReturn := make([]string, batchSize)
		for i := 0; i < batchSize; i++ {
			preimageBytes := make([]byte, 8)
			_, err := rand.Read(preimageBytes)
			if err != nil {
				http.Error(responseWriter, "500 internal server error", http.StatusInternalServerError)
				log.Printf("read random bytes failed: %v", err)
				return
			}
			preimage := base64.StdEncoding.EncodeToString(preimageBytes)
			difficultyBytes := make([]byte, int(math.Ceil(float64(difficultyLevel)/float64(8))))

			for j := 0; j < len(difficultyBytes); j++ {
				difficultyByte := byte(0)
				for k := 0; k < 8; k++ {
					currentBitIndex := (len(difficultyBytes) * 8) - (j*8 + k)
					if currentBitIndex > difficultyLevel {
						difficultyByte = difficultyByte | 1<<k
					}
				}
				difficultyBytes[j] = difficultyByte
			}

			difficulty := hex.EncodeToString(difficultyBytes)
			challenge := Challenge{
				Preimage:        preimage,
				Difficulty:      difficulty,
				DifficultyLevel: difficultyLevel,
			}
			challenge.CPUAndMemoryCost = scryptParameters.CPUAndMemoryCost
			challenge.BlockSize = scryptParameters.BlockSize
			challenge.Paralellization = scryptParameters.Paralellization
			challenge.KeyLength = scryptParameters.KeyLength

			challengeBytes, err := json.Marshal(challenge)
			if err != nil {
				http.Error(responseWriter, "500 internal server error", http.StatusInternalServerError)
				log.Printf("serialize challenge as json failed: %v", err)
				return
			}

			challengeBase64 := base64.StdEncoding.EncodeToString(challengeBytes)
			challenges[challengeBase64] = currentChallengesGeneration
			toReturn[i] = challengeBase64
		}
		toRemove := []string{}
		for k, generation := range challenges {
			if generation+deprecateAfterBatches < currentChallengesGeneration {
				toRemove = append(toRemove, k)
			}
		}
		for _, k := range toRemove {
			delete(challenges, k)
		}

		responseBytes, err := json.Marshal(toReturn)
		if err != nil {
			http.Error(responseWriter, "500 internal doodoo error", http.StatusInternalServerError)
			log.Printf("json marshal failed: %v", err)
			return
		}

		responseWriter.Write(responseBytes)
	})

	http.HandleFunc("/Verify", func(responseWriter http.ResponseWriter, request *http.Request) {

		if request.Method != "POST" {
			responseWriter.Header().Set("Allow", "POST")
			http.Error(responseWriter, "405 Method Not Allowed, try POST", http.StatusMethodNotAllowed)
		}

		requestQuery := request.URL.Query()
		challengeBase64 := requestQuery.Get("challenge")
		nonceHex := requestQuery.Get("nonce")

		if _, has := challenges[challengeBase64]; !has {
			http.Error(
				responseWriter,
				fmt.Sprintf(
					"404 challenge given by url param ?challenge=%s was not found",
					challengeBase64,
				),
				http.StatusNotFound,
			)
			return
		}

		delete(challenges, challengeBase64)

		nonceBuffer := make([]byte, 8)
		bytesWritten, err := hex.Decode(nonceBuffer, []byte(nonceHex))
		if nonceHex == "" || err != nil {
			http.Error(
				responseWriter,
				fmt.Sprintf(
					"400 bad request: nonce given by url param ?nonce=%s could not be hex decoded",
					nonceHex,
				),
				http.StatusBadRequest,
			)
			return
		}

		nonceBytes := nonceBuffer[:bytesWritten]

		challengeJson, err := base64.StdEncoding.DecodeString(challengeBase64)
		if err != nil {
			http.Error(responseWriter, "500 challenge couldn't be decoded", http.StatusInternalServerError)
			log.Printf("challenge %s couldn't be parsed: %v\n", challengeBase64, err)
			return
		}
		var challenge Challenge
		err = json.Unmarshal([]byte(challengeJson), &challenge)
		if err != nil {
			http.Error(responseWriter, "500 challenge couldn't be parsed", http.StatusInternalServerError)
			log.Printf("challenge %s (%s) couldn't be parsed: %v\n", challengeJson, challenge, err)
			return
		}

		preimageBytes := make([]byte, 8)
		n, err := base64.StdEncoding.Decode(preimageBytes, []byte(challenge.Preimage))
		if n != 8 || err != nil {
			http.Error(responseWriter, "500 invalid preimage", http.StatusInternalServerError)
			log.Printf("invalid preimage %s: %v\n", challenge.Preimage, err)
			return
		}

		hash, err := scrypt.Key(
			nonceBytes,
			preimageBytes,
			challenge.CPUAndMemoryCost,
			challenge.BlockSize,
			challenge.Paralellization,
			challenge.KeyLength,
		)

		if err != nil {
			http.Error(responseWriter, "500 scrypt returned error", http.StatusInternalServerError)
			log.Printf("scrypt returned error: %v\n", challengeJson, challenge, err)
			return
		}

		hashHex := hex.EncodeToString(hash)
		if hashHex[len(hashHex)-len(challenge.Difficulty):] > challenge.Difficulty {
			http.Error(
				responseWriter,
				fmt.Sprintf(
					"400 bad request: nonce given by url param ?nonce=%s did not result in a hash that meets the required difficulty",
					nonceHex,
				),
				http.StatusBadRequest,
			)
			return
		}

		responseWriter.WriteHeader(200)
		responseWriter.Write([]byte("OK"))
	})

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	log.Printf("💥  PoW! Captcha server listening on port %d", portNumber)

	err := http.ListenAndServe(fmt.Sprintf(":%d", portNumber), nil)

	// if got this far it means server crashed!
	panic(err)
}
