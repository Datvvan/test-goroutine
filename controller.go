package api

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
)

func VerifyDocument(c *gin.Context) {
	body := models.VerifyDocRequest{}
	user := models.User{}
	db := database.GetDB()
	if err := c.ShouldBind(&body); err != nil {
		utils.ResponseError(c, err, nil)
		return
	}
	err := utils.Validate(body)
	if err != nil {
		utils.ResponseError(c, err, nil)
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.ResponseError(c, err, nil)
		return
	}

	src, err := header.Open()
	if err != nil {
		utils.ResponseError(c, err, nil)
		return
	}
	defer src.Close()
	defer file.Close()

	buff := make([]byte, 512)
	_, err = file.Read(buff)
	if err != nil {
		utils.ResponseError(c, err, nil)
		return
	}
	filetype := http.DetectContentType(buff)
	if filetype != "application/pdf" {
		utils.ResponseError(c, errors.New("file is not pdf format"), nil)
		return
	}

	h := sha256.New()
	if _, err := io.Copy(h, src); err != nil {
		utils.ResponseError(c, err, nil)
		return
	}

	hashPdf := fmt.Sprintf("%x", h.Sum(nil))

	pubKey, err := models.GetDoc(hashPdf)
	if err != nil {
		utils.ResponseError(c, err, nil)
		return
	}
	if pubKey == models.DocumentNotFound {
		utils.ResponseError(c, errors.New("document can not found on verify system"), nil)
		return
	}
	errC := make(chan error)
	cerInfo := make(chan []map[string]string)

	go service.GetCertificate(c, cerInfo, errC)
	err = <-errC
	if err != nil {
		utils.ResponseError(c, err, nil)
		return
	}
	cer := <-cerInfo
	log.Println(cer)
	fmt.Printf("%s", cer)
	// cer, err := service.GetCertificate(c)
	// if err != nil {
	// 	utils.ResponseError(c, err, nil)
	// 	return
	// }
	err = db.Model(&user).Where("public_key=?", pubKey).Select()
	if err != nil {
		utils.ResponseNotFound(c, errors.New("have some wrong with user"))
		return
	}
	userCertificate := map[string]string{
		"countryName":         models.VietNamCountrySend,
		"stateOrProvinceName": models.DaNangStateDataSend,
		"organizationName":    models.MadisonCompanyDataSend,
		"commonName":          user.FullName,
		"emailAddress":        user.Email,
	}
	for _, v := range cer {
		if reflect.DeepEqual(userCertificate, v) {
			utils.ResponseSuccess(c, "Document verified", userCertificate)
			return
		}
	}

	utils.ResponseSuccess(c, "Document have not verified", nil)

}
