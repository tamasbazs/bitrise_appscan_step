package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

func main() {

	filePath := os.Getenv("APP_PATH")
	appName := os.Getenv("APP_NAME")
	usrLogin := os.Getenv("USER_LOGIN")
	password := os.Getenv("USER_PASSWORD")
	presence := os.Getenv("PRESENCE_ID")
	appUser := os.Getenv("APP_USER")
	appPassword := os.Getenv("APP_PASSWORD")

	if len(filePath) == 0 {
		fmt.Println("APK/IPA path is empty")
		os.Exit(10)
	}
	if len(appName) == 0 {
		fmt.Println("App name is empty")
		os.Exit(10)
	}
	if len(password) == 0 {
		fmt.Println("AppScan password is empty")
		os.Exit(10)
	}
	if len(usrLogin) == 0 {
		fmt.Println("AppScan username is empty")
		os.Exit(10)
	}

	client := &http.Client{}
	token, err := login(client, usrLogin, password)
	if err != nil {
		os.Exit(1)
	}

	idApp, err := findIDApp(client, token, appName)
	if err != nil {
		os.Exit(2)
	}

	idFile, err := uploadApp(client, token, filePath)
	if err != nil {
		os.Exit(4)
	}
	_, err = doScanMobile(client, appName, token, idFile, idApp, appUser, appPassword, presence)
	if err != nil {
		os.Exit(5)
	}

	fmt.Println("Terminating the application...")
}

func login(client *http.Client, usrLogin string, senha string) (map[string]string, error) {

	fmt.Println("Starting login...")
	m := make(map[string]string)

	jsonData := map[string]string{"Username": usrLogin, "Password": senha}
	jsonValue, _ := json.Marshal(jsonData)

	req, err := http.NewRequest("POST", "https://appscan.ibmcloud.com/api/V2/Account/IBMIdLogin", bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("Error creating a new HTTP request: %s\n", err)
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("The HTTP request to login failed with error: %s\n", err)
		return nil, err
	}

	data, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(data, &m)

	if m["Token"] == "" {
		fmt.Println("Not able to log in. Check your username and password.")
		return nil, errors.New("Not able to log in. Check your username and password")
	}

	fmt.Println("Exiting login...")
	return m, nil
}

func findIDApp(client *http.Client, token map[string]string, nameApp string) (string, error) {
	fmt.Println("Starting getting apps...")
	var apps []map[string]interface{}

	req, err := http.NewRequest("GET", "https://appscan.ibmcloud.com/api/V2/Apps", nil)
	if err != nil {
		fmt.Printf("Error creating a new HTTP request: %s\n", err)
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token["Token"])
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return "", nil
	}

	data, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(data, &apps)
	for _, app := range apps {
		if app["Name"] == nameApp {
			fmt.Println("App Id: ", app["Id"])
			id, _ := app["Id"].(string)
			return id, nil
		}
	}
	fmt.Println("No application found with the name " + nameApp + " in AppScan")
	return "", errors.New("No application found with the name " + nameApp + " in AppScan")
}
func uploadApp(client *http.Client, token map[string]string, filePath string) (string, error) {
	fmt.Println("Starting to upload files...")

	bodyBuffer := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuffer)

	fileWriter, err := bodyWriter.CreateFormFile("fileToUpload", filePath)
	if err != nil {
		fmt.Println("Error writting the request's body: ", err)
		return "", err
	}

	fileHandle, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening the APK/IPA file: ", err)
		return "", err
	}
	defer fileHandle.Close()

	_, err = io.Copy(fileWriter, fileHandle)
	if err != nil {
		fmt.Printf("Error copying the APK/IPA to the request's body: ", err)
		return "", err
	}

	err = bodyWriter.Close()

	req, err := http.NewRequest("POST", "https://appscan.ibmcloud.com/api/v2/FileUpload", bodyBuffer)
	if err != nil {
		fmt.Printf("Error creating a new request: %s\n", err)
		return "", err
	}

	// index do primeiro caracter do boundary
	startBoundary := strings.Index(bodyWriter.FormDataContentType(), "=") + 1

	//substring que forma o boundary
	boundary := bodyWriter.FormDataContentType()[startBoundary:]

	req.Header.Add("Content-Type", "multipart/form-data; boundary=\""+boundary+"\"")
	req.Header.Add("Accept-Encoding", "gzip,deflate")
	req.Header.Add("Accept", "text/plain")
	req.Header.Add("Connection", "Keep-Alive")
	req.Header.Add("Authorization", "Bearer "+token["Token"])

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return "", err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading the response data %s\n", err)
		return "", err
	}

	resp.Body.Close()
	responseData := make(map[string]string)
	json.Unmarshal(data, &responseData)

	if resp.StatusCode != 201 {
		fmt.Println("The HTTP request failed with status " + string(resp.StatusCode) + ": " + responseData["Message"])
		return "", errors.New("The HTTP request failed with status " + string(resp.StatusCode) + ": " + responseData["Message"])
	}

	fmt.Println("Upload Succesful...")
	return responseData["FileId"], nil
}

func doScanMobile(client *http.Client, name string, token map[string]string, idFile string, idApp string, login string, senha string, presence string) (map[string]string, error) {
	fmt.Println("Starting scan...")

	m := make(map[string]string)

	jsonData := map[string]string{
		"ApplicationFileId":      idFile,
		"LoginUser":              login,
		"LoginPassword":          senha,
		"ExtraField":             "",
		"PresenceId":             presence,
		"ScanName":               name,
		"EnableMailNotification": "false",
		"Locale":                 "en-US",
		"AppId":                  idApp,
		"Execute":                "true",
		"Personal":               "false",
		"OfferingType":           "None"}

	jsonValue, _ := json.Marshal(jsonData)

	req, err := http.NewRequest("POST", "https://appscan.ibmcloud.com/api/v2/Scans/MobileAnalyzer", bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("Error creating a new request: %s\n", err)
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token["Token"])
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return nil, err
	}

	data, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(data, &m)
	if resp.StatusCode != 201 {
		fmt.Println("The HTTP request failed with status " + string(resp.StatusCode) + ": " + m["Message"])
		return nil, errors.New("The HTTP request failed with status " + string(resp.StatusCode) + ": " + m["Message"])
	}

	return m, nil
}