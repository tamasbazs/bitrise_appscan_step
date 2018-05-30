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

	var filename = os.Getenv("BITRISE_APK_PATH")
	appName := os.Getenv("APP_NAME")
	usrLogin := os.Getenv("USER_LOGIN")
	senha := os.Getenv("USER_PASSWORD")

	client := &http.Client{}
	token, err := login(client, usrLogin, senha)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	idApp, err := findIDApp(client, token, appName)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	idFile, err := uploadApp(client, token, filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
	}

	_, err = doScanMobile(client, appName, token, idFile, idApp, usrLogin, senha)
	if err != nil {
		fmt.Println(err)
		os.Exit(4)
	}

	fmt.Println("Terminating the application...")
}

func login(client *http.Client, usrLogin string, senha string) (map[string]string, error) {

	fmt.Println("Starting login......")
	m := make(map[string]string)

	jsonData := map[string]string{"Username": usrLogin, "Password": senha}
	jsonValue, _ := json.Marshal(jsonData)

	req, err := http.NewRequest("POST", "https://appscan.ibmcloud.com/api/V2/Account/IBMIdLogin", bytes.NewBuffer(jsonValue))
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return nil, err
	}

	data, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(data, &m)

	return m, nil
}

func findIDApp(client *http.Client, token map[string]string, nomeApp string) (string, error) {
	fmt.Println("Starting getting apps.....")
	var retorno []map[string]interface{}

	req, err := http.NewRequest("GET", "https://appscan.ibmcloud.com/api/V2/Apps", nil)
	req.Header.Set("Authorization", "Bearer "+token["Token"])
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return "", nil
	}

	data, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(data, &retorno)
	for _, app := range retorno {
		if app["Name"] == nomeApp {
			fmt.Println("ID do App ", app["Id"])
			id, _ := app["Id"].(string)
			return id, nil
		}
	}

	return "", errors.New("Nenhuma aplicação com o nome " + nomeApp + " foi encontrada no AppScan")
}

func uploadApp(client *http.Client, token map[string]string, filename string) (string, error) {
	fmt.Println("Starting to upload files...")

	bodyBuffer := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuffer)

	fileWriter, err := bodyWriter.CreateFormFile("fileToUpload", filename)
	if err != nil {
		fmt.Println("error writing to buffer")
		return "", err
	}

	fileHandle, err := os.Open(filename)
	if err != nil {
		fmt.Printf("error opening file")
		return "", err
	}
	defer fileHandle.Close()

	_, err = io.Copy(fileWriter, fileHandle)
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return "", err
	}

	err = bodyWriter.Close()

	req, err := http.NewRequest("POST", "https://appscan.ibmcloud.com/api/v2/FileUpload", bodyBuffer)
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return "", err
	}

	// index do primeiro caracter do boundary
	inicioBoundary := strings.Index(bodyWriter.FormDataContentType(), "=") + 1

	//substring que forma o boundary
	boundary := bodyWriter.FormDataContentType()[inicioBoundary:]

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
		fmt.Println("Imprimindo erro")
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return "", err
	}
	resp.Body.Close()
	retorno := make(map[string]string)
	json.Unmarshal(data, &retorno)

	fmt.Println("Exiting upload\n ...................................")
	return retorno["FileId"], nil
}

func doScanMobile(client *http.Client, name string, token map[string]string, idFile string, idApp string, usrLogin string, senha string) (map[string]string, error) {
	fmt.Println("Starting scan......")

	m := make(map[string]string)

	jsonData := map[string]string{
		"ApplicationFileId":      idFile,
		"LoginUser":              usrLogin,
		"LoginPassword":          senha,
		"ExtraField":             "",
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
	return m, nil
}