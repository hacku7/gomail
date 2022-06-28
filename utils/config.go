/**
* @author: [hk7]
* @Data:   17:58
 */

package utils

import (
	"os"
	"path/filepath"
)

type Config struct {
	EmailSettings
}

type EmailSettings struct {
	EnableSignUpWithEmail             bool
	EnableSignInWithEmail             *bool
	EnableSignInWithUsername          *bool
	FeedbackName                      string
	FeedbackEmail                     string
	FeedbackOrganization              *string
	EnableSMTPAuth                    *bool
	SMTPUsername                      string
	SMTPPassword                      string
	SMTPServer                        string
	SMTPPort                          string
	ConnectionSecurity                string
	SkipServerCertificateVerification *bool
	TempSubject                       string
	TempTitle                         string
	TempInfo                          string
	TempName                          string
}

func FindDir(dir string) (string, bool) {
	fileName := "."
	found := false
	if _, err := os.Stat("./" + dir + "/"); err == nil {
		fileName, _ = filepath.Abs("./" + dir + "/")
		found = true
	} else if _, err := os.Stat("../" + dir + "/"); err == nil {
		fileName, _ = filepath.Abs("../" + dir + "/")
		found = true
	} else if _, err := os.Stat("../../" + dir + "/"); err == nil {
		fileName, _ = filepath.Abs("../../" + dir + "/")
		found = true
	}

	return fileName + "/", found
}
