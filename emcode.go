/**
* @author: [hk7]
* @Data:   16:23
 */

package main

import (
	"github.com/hacku7/gomail/codegen"
	"github.com/hacku7/gomail/utils"
)

func main() {
	var cfg *utils.Config = &utils.Config{}
	err := codegen.LoadConfig(cfg, "config.json")
	if err != nil {
	}
	code, err := codegen.NewEmailSender("mail@qq.com", cfg.TempSubject, "templates/verify_body.html", cfg)
	if err != nil {

	}
	print(code)
}
