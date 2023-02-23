package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetCertificate(c *gin.Context, cer chan []map[string]string, errC chan error) {

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		errC <- err
		return
	}

	defer close(cer)
	defer close(errC)

	src, err := header.Open()
	if err != nil {
		errC <- err
		return
	}
	defer src.Close()
	defer file.Close()

	buff := make([]byte, 512)
	_, err = file.Read(buff)
	if err != nil {
		errC <- err
		return
	}

	filetype := http.DetectContentType(buff)
	if filetype != "application/pdf" {
		err = errors.New("file is not pdf format")
		errC <- err
		return
	}

	fw, err := w.CreateFormFile("file", header.Filename)
	if err != nil {
		errC <- err
		return
	}
	_, err = io.Copy(fw, src)
	if err != nil {
		errC <- err
		return
	}

	err = w.Close()
	if err != nil {
		return
	}

	verifySvcConfig, err := config.LoadVerifySvcConfig(".")
	if err != nil {
		errC <- err
		return
	}

	req, err := http.NewRequest("POST", verifySvcConfig.VerifyServiceURL, &b)
	if err != nil {
		errC <- err
		return
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0aW1lX3N0YW1wIjoxNjc3MDQ5MjEyLCJ1c2VyX2lkIjoiNThiYjg1ZmYtMzk2Yi00OTM0LWJhYjgtYTM3NTQzMzEwMjk5In0.xilbfkbekHrKBag3qSpQ9QVVZ8YBy6Nf9nJxJEJFp3I")

	// Submit the request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		errC <- err
		return
	}

	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
		errC <- err
		return
	}

	resp := VerifyResponse{}
	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return
	}
	log.Println(resp.Data)
	cer <- resp.Data
}
