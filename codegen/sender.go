/**
* @author: [hk7]
* @Data:   20:02
 */

package codegen

import (
	"errors"
	"fmt"
	l4g "github.com/alecthomas/log4go"
	"github.com/hacku7/gomail/utils"
	"github.com/nicksnyder/go-i18n/i18n"
	"github.com/spf13/viper"
	"gopkg.in/gomail.v2"
	"html/template"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

var (
	cfgMutex      = &sync.Mutex{}
	CfgFileName   string
	T             i18n.TranslateFunc
	htmlTemplates *template.Template
)

func NewEmailSender(userEmail, subject, TempBody string, cfg *utils.Config) (int, error) {
	if len(userEmail) == 0 || len(subject) == 0 || len(TempBody) == 0 {
		return 0, errors.New("api.new.email.sender.error")
	}

	err := utils.InitHTML()
	if err != nil {
		return 0, err
	}
	verifyCode, err := initVerifyCode(userEmail, subject, TempBody, cfg)
	if err != nil {
		return 0, err
	}

	return verifyCode, nil
}

func MakeVerifyCode() (emailCode int, code string) {
	emailCode = rand.New(rand.NewSource(time.Now().UnixNano())).Intn(899999) + 100000
	code = strconv.Itoa(emailCode)
	return emailCode, code
}

func LoadConfig(cfg *utils.Config, fileName string) error {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	fileNameWithExtension := filepath.Base(fileName)
	fileExtension := filepath.Ext(fileNameWithExtension)
	fileDir := filepath.Dir(fileName)

	if len(fileNameWithExtension) > 0 {
		fileNameOnly := fileNameWithExtension[:len(fileNameWithExtension)-len(fileExtension)]
		viper.SetConfigName(fileNameOnly)
	} else {
		viper.SetConfigName("config")
	}

	if len(fileDir) > 0 {
		viper.AddConfigPath(fileDir)
	}

	viper.SetConfigType("json")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("../config")
	viper.AddConfigPath("../../config")
	viper.AddConfigPath(".")

	configReadErr := viper.ReadInConfig()
	if configReadErr != nil {
		errMsg := T("utils.config.load_config.opening.panic", map[string]interface{}{"Filename": fileName, "Error": configReadErr.Error()})
		fmt.Fprintln(os.Stderr, errMsg)
		return configReadErr
	}

	unmarshalErr := viper.Unmarshal(cfg)
	if unmarshalErr != nil {
		errMsg := T("utils.config.load_config.decoding.panic", map[string]interface{}{"Filename": fileName, "Error": unmarshalErr.Error()})
		fmt.Fprintln(os.Stderr, errMsg)
		return unmarshalErr
	}

	//CfgFileName = viper.ConfigFileUsed()
	//CfgJson, _ := json.Marshal(config)
	//fmt.Sprintf("%x", md5.Sum(CfgJson))

	return nil
}

// SendVerifyCode init send
func initVerifyCode(userEmail, subject, locale string, cfg *utils.Config) (int, error) {
	T := utils.GetUserTranslations(locale)
	emailCode, code := MakeVerifyCode()

	bodyPage := utils.NewHTMLTemplate(cfg.TempName, locale)
	bodyPage.Props["Title"] = T(cfg.TempTitle, map[string]interface{}{"SiteName": ""})
	bodyPage.Props["Info"] = T(cfg.TempInfo)
	bodyPage.Props["Code"] = code

	message := "\r\n<html><body>" + bodyPage.Render() + "</body></html>"

	return emailCode, sendVerifyCode(userEmail, subject, message, cfg)
}

// sendVerifyCode 邮件发送
func sendVerifyCode(userEmail, subject, body string, cfg *utils.Config) (err error) {
	m := gomail.NewMessage()
	m.SetHeader("From", cfg.FeedbackEmail)
	m.SetHeader("To", userEmail)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	port, err := strconv.Atoi(cfg.SMTPPort)
	mailer := gomail.NewDialer(cfg.SMTPServer, port, cfg.SMTPUsername, cfg.SMTPPassword)

	if err = mailer.DialAndSend(m); err != nil {
		l4g.Error(err)
	}
	return err
}
