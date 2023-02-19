package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	config Config
)

type Config struct {
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
	Token  string `json:"token"`
	Path   string `json:"path"`
}

type Content struct {
	DownloadUrl string `json:"download_url"`
}

type Response struct {
	Content Content `json:"content"`
}

func getFile(filePath string) (string, error) {
	re := regexp.MustCompile(`^https?://`)
	if re.MatchString(filePath) {
		resp, err := http.Get(filePath)
		if err != nil {
			return "", err
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("download failed: %s", filePath)
		}

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		return base64.StdEncoding.EncodeToString(b), nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	fileInfo, err := file.Stat()
	if err != nil {
		return "", err
	}

	fileBytes := make([]byte, fileInfo.Size())
	_, err = file.Read(fileBytes)

	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(fileBytes), nil
}

func upload(message string, filename string, fileBase64Content string) (string, error) {
	// https://docs.github.com/en/rest/repos/contents?apiVersion=2022-11-28#create-or-update-file-contents

	apiUrl := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s/%s",
		strings.Trim(config.Repo, "/"), strings.Trim(config.Path, "/"), filename)

	values := map[string]string{
		"message": message,
		"content": fileBase64Content,
		"branch":  config.Branch,
	}

	payload, err := json.Marshal(values)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("PUT", apiUrl, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.Token))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("ret code error: %d", resp.StatusCode)
	}

	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	t := &Response{}
	err = json.Unmarshal(all, t)
	if err != nil {
		return "", err
	}

	return t.Content.DownloadUrl, err
}

func getExt(buffer []byte) string {
	if len(buffer) > 0 {
		contentType := http.DetectContentType(buffer)
		ext, _ := mime.ExtensionsByType(contentType)
		if len(ext) > 0 {
			return ext[0]
		}
	}
	return ""
}
func processUpload(imageList []string) {
	if len(imageList) == 0 {
		fmt.Println("no image url detected")
		os.Exit(1)
	}

	retList := make([]string, 0)
	for _, img := range imageList {
		filename := filepath.Base(img)
		base64String, err := getFile(img)
		if err != nil {
			fmt.Printf("Failed to get file: %v", err)
			continue
		}

		if base64String == "" {
			fmt.Printf("base64String empty: %v", img)
			continue
		} else {
			if !strings.Contains(filename, ".") {
				buffer, _ := base64.StdEncoding.DecodeString(base64String)
				filename += getExt(buffer)
			}
			filename = time.Now().Format("20060102150405") + filename
			downloadUrl, err1 := upload("upload file", filename, base64String)
			if err1 != nil {
				fmt.Printf("Failed to upload: %v", err)
				continue
			}
			retList = append(retList, downloadUrl)
		}
	}

	if len(retList) > 0 {
		res := "Upload Success:\n" + strings.Join(retList, "\n")
		fmt.Print(res)
	} else {
		fmt.Print("none upload success")
	}
}

func loadConfig() error {
	var configFile string
	flag.StringVar(&configFile, "f", "", `
-f path/to/config.json
default ./config.json

config file content struct example:
{
	"repo" : "owner/projectName",
	"branch": "main",
	"token": "access token",
	"path": "image/2023"
}
`)
	flag.Parse()
	if configFile == "" {
		abs, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return err
		}
		configFile = path.Join(abs, "config.json")
	}
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	buffer, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(buffer, &config)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	if err := loadConfig(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	processUpload(flag.Args())
}
