package service

import (
	"fmt"
	"rustdesk-api-server-pro/app/model"
	"rustdesk-api-server-pro/config"
	"rustdesk-api-server-pro/db"
	"strings"

	mail "github.com/xhit/go-simple-mail/v2"
)

type EmailService struct {
	mailer *mail.SMTPServer
	config *config.ServerConfig
}

var service *EmailService

func NewEmailService() *EmailService {

	// 单例模式
	if service != nil {
		return service
	}

	config := config.GetServerConfig()

	mailer := mail.NewSMTPClient()

	mailer.Host = config.SmtpConfig.Host
	mailer.Port = config.SmtpConfig.Port
	mailer.Username = config.SmtpConfig.Username
	mailer.Password = config.SmtpConfig.Password
	switch config.SmtpConfig.Encryption {
	case "ssl/tls":
		mailer.Encryption = mail.EncryptionSSLTLS
	case "starttls":
		mailer.Encryption = mail.EncryptionSTARTTLS
	default:
		mailer.Encryption = mail.EncryptionNone
	}

	return &EmailService{
		mailer: mailer,
		config: config,
	}
}

func (service *EmailService) Send(userId, tplId int, to string, vars map[string]string) error {

	sendLog := &model.EmailLogs{
		UserId: userId,
		TplId:  tplId,
		From:   service.config.SmtpConfig.From,
		To:     to,
	}

	var template model.MailTemplate
	get, err := db.DbEngine.Where("id = ?", tplId).Get(&template)
	if err != nil || !get {
		sendLog.Status = model.MAIL_SEND_ERR
		sendLog.Logs = fmt.Sprintf("template not found or error: %s", err.Error())
		db.DbEngine.Insert(sendLog)
		return err
	}

	body := template.Contents
	for k, v := range vars {
		body = strings.Replace(body, k, v, -1)
	}

	message := mail.NewMSG()
	message.SetFrom(service.config.SmtpConfig.From)
	message.AddTo(to)
	message.SetSubject(template.Subject)
	message.SetBody(mail.TextHTML, body)

	sender, err := service.mailer.Connect()
	if err != nil {
		sendLog.Status = model.MAIL_SEND_ERR
		sendLog.Logs = fmt.Sprintf("can not connect smtp server error: %s", err.Error())
		db.DbEngine.Insert(sendLog)
		return err
	}
	err = message.Send(sender)
	if err != nil {
		sendLog.Status = model.MAIL_SEND_ERR
		sendLog.Logs = fmt.Sprintf("send error: %s", err.Error())
		db.DbEngine.Insert(sendLog)
		return err
	}

	sendLog.Subject = template.Subject
	sendLog.Contents = body
	sendLog.Status = model.MAIL_SEND_OK

	db.DbEngine.Insert(sendLog)

	return nil
}
