package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	errors "git.sequentialread.com/forest/pkg-errors"
	"golang.org/x/crypto/scrypt"
)

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

var currentChallengesGeneration = map[string]int{}
var challenges = map[string]map[string]int{}

func main() {

	var err error

	batchSize := 1000
	deprecateAfterBatches := 10
	portNumber := 2370
	scryptCPUAndMemoryCost := 4096
	batchSizeEnv := os.ExpandEnv("$POW_CAPTCHA_BATCH_SIZE")
	deprecateAfterBatchesEnv := os.ExpandEnv("$POW_CAPTCHA_DEPRECATE_AFTER_BATCHES")
	portNumberEnv := os.ExpandEnv("$POW_CAPTCHA_LISTEN_PORT")
	scryptCPUAndMemoryCostEnv := os.ExpandEnv("$POW_CAPTCHA_SCRYPT_CPU_AND_MEMORY_COST")
	if batchSizeEnv != "" {
		batchSize, err = strconv.Atoi(batchSizeEnv)
		if err != nil {
			panic(errors.Wrapf(err, "can't start the app because the POW_CAPTCHA_BATCH_SIZE '%s' can't be converted to an integer", batchSizeEnv))
		}
	}
	if deprecateAfterBatchesEnv != "" {
		deprecateAfterBatches, err = strconv.Atoi(deprecateAfterBatchesEnv)
		if err != nil {
			panic(errors.Wrapf(err, "can't start the app because the POW_CAPTCHA_DEPRECATE_AFTER_BATCHES '%s' can't be converted to an integer", deprecateAfterBatchesEnv))
		}
	}
	if portNumberEnv != "" {
		portNumber, err = strconv.Atoi(portNumberEnv)
		if err != nil {
			panic(errors.Wrapf(err, "can't start the app because the POW_CAPTCHA_LISTEN_PORT '%s' can't be converted to an integer", portNumberEnv))
		}
	}
	if scryptCPUAndMemoryCostEnv != "" {
		scryptCPUAndMemoryCost, err = strconv.Atoi(scryptCPUAndMemoryCostEnv)
		if err != nil {
			panic(errors.Wrapf(err, "can't start the app because the POW_CAPTCHA_SCRYPT_CPU_AND_MEMORY_COST '%s' can't be converted to an integer", scryptCPUAndMemoryCostEnv))
		}
	}

	apiTokensFolder := locateAPITokensFolder()
	adminAPIToken := os.ExpandEnv("$POW_CAPTCHA_ADMIN_API_TOKEN")
	if adminAPIToken == "" {
		panic(errors.New("can't start the app, the POW_CAPTCHA_ADMIN_API_TOKEN environment variable is required"))
	}

	scryptParameters := ScryptParameters{
		CPUAndMemoryCost: scryptCPUAndMemoryCost,
		BlockSize:        8,
		Paralellization:  1,
		KeyLength:        16,
	}

	requireMethod := func(method string) func(http.ResponseWriter, *http.Request) bool {
		return func(responseWriter http.ResponseWriter, request *http.Request) bool {
			if request.Method != method {
				responseWriter.Header().Set("Allow", method)
				http.Error(responseWriter, fmt.Sprintf("405 Method Not Allowed, try %s", method), http.StatusMethodNotAllowed)
				return true
			}
			return false
		}
	}

	requireAdmin := func(responseWriter http.ResponseWriter, request *http.Request) bool {
		if request.Header.Get("Authorization") != fmt.Sprintf("Bearer %s", adminAPIToken) {
			http.Error(responseWriter, "401 Unauthorized", http.StatusUnauthorized)
			return true
		}
		return false
	}

	requireToken := func(responseWriter http.ResponseWriter, request *http.Request) bool {
		authorizationHeader := request.Header.Get("Authorization")
		if !strings.HasPrefix(authorizationHeader, "Bearer ") {
			http.Error(responseWriter, "401 Unauthorized: Authorization header is required and must start with 'Bearer '", http.StatusUnauthorized)
			return true
		}
		token := strings.TrimPrefix(authorizationHeader, "Bearer ")
		if token == "" {
			http.Error(responseWriter, "401 Unauthorized: Authorization Bearer token is required", http.StatusUnauthorized)
			return true
		}
		if !regexp.MustCompile("^[0-9a-f]{32}$").MatchString(token) {
			errorMsg := fmt.Sprintf("401 Unauthorized: Authorization Bearer token '%s' must be a 32 character hex string", token)
			http.Error(responseWriter, errorMsg, http.StatusUnauthorized)
			return true
		}
		fileInfos, err := ioutil.ReadDir(apiTokensFolder)
		if err != nil {
			log.Printf("failed to list the apiTokensFolder (%s): %v", apiTokensFolder, err)
			http.Error(responseWriter, "500 internal server error", http.StatusInternalServerError)
			return true
		}
		foundToken := false
		for _, fileInfo := range fileInfos {
			if strings.HasPrefix(fileInfo.Name(), token) {
				foundToken = true
				break
			}
		}
		if !foundToken {
			errorMsg := fmt.Sprintf("401 Unauthorized: Authorization Bearer token '%s' was in the right format, but it was unrecognized", token)
			http.Error(responseWriter, errorMsg, http.StatusUnauthorized)
			return true
		}
		return false
	}

	myHTTPHandleFunc("/Tokens", requireMethod("GET"), requireAdmin, func(responseWriter http.ResponseWriter, request *http.Request) bool {
		fileInfos, err := ioutil.ReadDir(apiTokensFolder)
		if err != nil {
			log.Printf("failed to list the apiTokensFolder (%s): %v", apiTokensFolder, err)
			http.Error(responseWriter, "500 internal server error", http.StatusInternalServerError)
			return true
		}

		output := []string{}

		for _, fileInfo := range fileInfos {
			filenameSplit := strings.Split(fileInfo.Name(), "_")
			if len(filenameSplit) == 2 {
				filepath := path.Join(apiTokensFolder, fileInfo.Name())
				content, err := ioutil.ReadFile(filepath)
				if err != nil {
					log.Printf("failed to read the token file (%s): %v", filepath, err)
					http.Error(responseWriter, "500 internal server error", http.StatusInternalServerError)
					return true
				}
				contentInt64, err := strconv.ParseInt(string(content), 10, 64)
				timestampString := time.Unix(contentInt64, 0).UTC().Format(time.RFC3339)
				output = append(output, fmt.Sprintf("%s,%s,%d,%s", filenameSplit[0], filenameSplit[1], contentInt64, timestampString))
			}

		}

		responseWriter.Header().Set("Content-Type", "text/plain")
		responseWriter.Write([]byte(strings.Join(output, "\n")))

		return true
	})

	myHTTPHandleFunc("/Tokens/Create", requireMethod("POST"), requireAdmin, func(responseWriter http.ResponseWriter, request *http.Request) bool {
		name := request.URL.Query().Get("name")
		if name == "" {
			http.Error(responseWriter, "400 Bad Request: url param ?name=<string> is required", http.StatusBadRequest)
			return true
		}
		// we use underscore as a syntax character in the filename, so we have to remove it from the user-inputted name
		name = strings.ReplaceAll(name, "_", "-")
		// let's also remove any sort of funky or path-related characters
		name = strings.ReplaceAll(name, "*", "")
		name = strings.ReplaceAll(name, "?", "")
		name = strings.ReplaceAll(name, "/", "-")
		name = strings.ReplaceAll(name, "\\", "-")
		name = strings.ReplaceAll(name, ".", "-")

		tokenBytes := make([]byte, 16)
		rand.Read(tokenBytes)

		ioutil.WriteFile(
			path.Join(apiTokensFolder, fmt.Sprintf("%x_%s", tokenBytes, name)),
			[]byte(fmt.Sprintf("%d", time.Now().Unix())),
			0644,
		)

		fmt.Fprintf(responseWriter, "%x", tokenBytes)

		return true
	})

	myHTTPHandleFunc("/Tokens/Revoke", requireMethod("POST"), requireAdmin, func(responseWriter http.ResponseWriter, request *http.Request) bool {
		token := request.URL.Query().Get("token")
		if token == "" {
			http.Error(responseWriter, "400 Bad Request: url param ?token=<string> is required", http.StatusBadRequest)
			return true
		}
		if !regexp.MustCompile("^[0-9a-f]{32}$").MatchString(token) {
			errorMsg := fmt.Sprintf("400 Bad Request: url param ?token=%s must be a 32 character hex string", token)
			http.Error(responseWriter, errorMsg, http.StatusBadRequest)
			return true
		}

		fileInfos, err := ioutil.ReadDir(apiTokensFolder)
		if err != nil {
			log.Printf("failed to list the apiTokensFolder (%s): %v", apiTokensFolder, err)
			http.Error(responseWriter, "500 internal server error", http.StatusInternalServerError)
			return true
		}
		for _, fileInfo := range fileInfos {
			if strings.HasPrefix(fileInfo.Name(), token) {
				os.Remove(path.Join(apiTokensFolder, fileInfo.Name()))
			}
		}

		responseWriter.Write([]byte("Revoked"))
		return true
	})

	myHTTPHandleFunc("/GetChallenges", requireMethod("POST"), requireToken, func(responseWriter http.ResponseWriter, request *http.Request) bool {

		// requireToken already validated the API Token, so we can just do this:
		token := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")

		if _, has := currentChallengesGeneration[token]; !has {
			currentChallengesGeneration[token] = 0
		}
		if _, has := challenges[token]; !has {
			challenges[token] = map[string]int{}
		}
		currentChallengesGeneration[token]++

		requestQuery := request.URL.Query()
		difficultyLevelString := requestQuery.Get("difficultyLevel")
		difficultyLevel, err := strconv.Atoi(difficultyLevelString)
		if err != nil {
			errorMessage := fmt.Sprintf(
				"400 url param ?difficultyLevel=%s value could not be converted to an integer",
				difficultyLevelString,
			)
			http.Error(responseWriter, errorMessage, http.StatusBadRequest)
			return true
		}

		toReturn := make([]string, batchSize)
		for i := 0; i < batchSize; i++ {
			preimageBytes := make([]byte, 8)
			_, err := rand.Read(preimageBytes)
			if err != nil {
				log.Printf("read random bytes failed: %v", err)
				http.Error(responseWriter, "500 internal server error", http.StatusInternalServerError)
				return true
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
				log.Printf("serialize challenge as json failed: %v", err)
				http.Error(responseWriter, "500 internal server error", http.StatusInternalServerError)
				return true
			}

			challengeBase64 := base64.StdEncoding.EncodeToString(challengeBytes)
			challenges[token][challengeBase64] = currentChallengesGeneration[token]
			toReturn[i] = challengeBase64
		}
		toRemove := []string{}
		for k, generation := range challenges[token] {
			if generation+deprecateAfterBatches < currentChallengesGeneration[token] {
				toRemove = append(toRemove, k)
			}
		}
		for _, k := range toRemove {
			delete(challenges[token], k)
		}

		responseBytes, err := json.Marshal(toReturn)
		if err != nil {
			log.Printf("json marshal failed: %v", err)
			http.Error(responseWriter, "500 internal server error", http.StatusInternalServerError)
			return true
		}

		responseWriter.Write(responseBytes)

		return true
	})

	myHTTPHandleFunc("/Verify", requireMethod("POST"), requireToken, func(responseWriter http.ResponseWriter, request *http.Request) bool {

		// requireToken already validated the API Token, so we can just do this:
		token := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")

		requestQuery := request.URL.Query()
		challengeBase64 := requestQuery.Get("challenge")
		nonceHex := requestQuery.Get("nonce")

		_, hasAnyChallenges := challenges[token]
		hasChallenge := false
		if hasAnyChallenges {
			_, hasChallenge = challenges[token][challengeBase64]
		}

		if !hasChallenge {
			errorMessage := fmt.Sprintf("404 challenge given by url param ?challenge=%s was not found", challengeBase64)
			http.Error(responseWriter, errorMessage, http.StatusNotFound)
			return true
		}

		delete(challenges[token], challengeBase64)

		nonceBuffer := make([]byte, 8)
		bytesWritten, err := hex.Decode(nonceBuffer, []byte(nonceHex))
		if nonceHex == "" || err != nil {
			errorMessage := fmt.Sprintf("400 bad request: nonce given by url param ?nonce=%s could not be hex decoded", nonceHex)
			http.Error(responseWriter, errorMessage, http.StatusBadRequest)
			return true
		}

		nonceBytes := nonceBuffer[:bytesWritten]

		challengeJSON, err := base64.StdEncoding.DecodeString(challengeBase64)
		if err != nil {
			log.Printf("challenge %s couldn't be parsed: %v\n", challengeBase64, err)
			http.Error(responseWriter, "500 challenge couldn't be decoded", http.StatusInternalServerError)
			return true
		}
		var challenge Challenge
		err = json.Unmarshal([]byte(challengeJSON), &challenge)
		if err != nil {
			log.Printf("challenge %s (%s) couldn't be parsed: %v\n", string(challengeJSON), challengeBase64, err)
			http.Error(responseWriter, "500 challenge couldn't be parsed", http.StatusInternalServerError)
			return true
		}

		preimageBytes := make([]byte, 8)
		n, err := base64.StdEncoding.Decode(preimageBytes, []byte(challenge.Preimage))
		if n != 8 || err != nil {
			log.Printf("invalid preimage %s: %v\n", challenge.Preimage, err)
			http.Error(responseWriter, "500 invalid preimage", http.StatusInternalServerError)
			return true
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
			log.Printf("scrypt returned error: %v\n", err)
			http.Error(responseWriter, "500 scrypt returned error", http.StatusInternalServerError)
			return true
		}

		hashHex := hex.EncodeToString(hash)
		if hashHex[len(hashHex)-len(challenge.Difficulty):] > challenge.Difficulty {
			errorMessage := fmt.Sprintf(
				"400 bad request: nonce given by url param ?nonce=%s did not result in a hash that meets the required difficulty",
				nonceHex,
			)
			http.Error(responseWriter, errorMessage, http.StatusBadRequest)
			return true
		}

		responseWriter.WriteHeader(200)
		responseWriter.Write([]byte("OK"))
		return true
	})

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	log.Printf("💥  PoW! Captcha server listening on port %d", portNumber)

	err = http.ListenAndServe(fmt.Sprintf(":%d", portNumber), nil)

	// if got this far it means server crashed!
	panic(err)
}

func myHTTPHandleFunc(path string, stack ...func(http.ResponseWriter, *http.Request) bool) {
	http.HandleFunc(path, func(responseWriter http.ResponseWriter, request *http.Request) {
		for _, handler := range stack {
			if handler(responseWriter, request) {
				break
			}
		}
	})
}

func locateAPITokensFolder() string {
	workingDirectory, err := os.Getwd()
	if err != nil {
		log.Fatalf("locateAPITokensFolder(): can't os.Getwd(): %v", err)
	}
	executableDirectory, err := getCurrentExecDir()
	if err != nil {
		log.Fatalf("locateAPITokensFolder(): can't getCurrentExecDir(): %v", err)
	}

	nextToExecutable := filepath.Join(executableDirectory, "PoW_Captcha_API_Tokens")
	inWorkingDirectory := filepath.Join(workingDirectory, "PoW_Captcha_API_Tokens")

	nextToExecutableStat, err := os.Stat(nextToExecutable)
	foundKeysNextToExecutable := err == nil && nextToExecutableStat.IsDir()
	inWorkingDirectoryStat, err := os.Stat(inWorkingDirectory)
	foundKeysInWorkingDirectory := err == nil && inWorkingDirectoryStat.IsDir()
	if foundKeysNextToExecutable && foundKeysInWorkingDirectory && workingDirectory != executableDirectory {
		log.Fatalf(`locateAPITokensFolder(): Something went wrong with your installation, 
			I found two PoW_Captcha_API_Tokens folders and I'm not sure which one to use.
			One of them is located at %s
			and the other is at %s`, inWorkingDirectory, nextToExecutable)
	}
	if foundKeysInWorkingDirectory {
		return inWorkingDirectory
	} else if foundKeysNextToExecutable {
		return nextToExecutable
	}

	log.Fatalf(`locateAPITokensFolder(): I didn't find a PoW_Captcha_API_Tokens folder 
		in the current working directory (in %s) or next to the executable (in %s)`, workingDirectory, executableDirectory)

	return ""
}

func getCurrentExecDir() (dir string, err error) {
	path, err := exec.LookPath(os.Args[0])
	if err != nil {
		fmt.Printf("exec.LookPath(%s) returned %s\n", os.Args[0], err)
		return "", err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("filepath.Abs(%s) returned %s\n", path, err)
		return "", err
	}

	dir = filepath.Dir(absPath)

	return dir, nil
}
