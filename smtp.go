package main

import (
	"bytes"
	"crypto/tls"
	"html/template"
	"log"
	"time"

	mail "github.com/xhit/go-simple-mail/v2"
)

func smtpProducer(toA []string, alertType string, subject string, text string) {
	type EMailContent struct {
		Tittle    string
		Text      string
		AlertType string
	}

	log.Print("Sending EMail alerts to QUEUE...")

	tmpl, err := template.ParseFiles("email.tmpl")
	tmplContent := EMailContent{ // Structure for filling EMail template
		Tittle:    subject,
		Text:      text,
		AlertType: alertType,
	}
	if err != nil {
		log.Print("Not a successful open template EMail messages (", err, ")")
	} else {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, tmplContent)

		if err != nil {
			log.Print("Not a successful execute template EMail messages (", err, ")")
		} else {
			msg := SMTPMessageT{
				ToA:     toA,
				Subject: subject,
				Body:    buf.String(),
			}
			select { // Non-Blocking sending messages
			case SMTPQueue <- msg:
			default:
				log.Print("SMTP QUEUE is filled, the message is discarded")
			}
		}
	}
}

func smtpConsumer() {
	if AppConfig.SMTP.ENABLED {
		log.Printf("START SMTP CONSUMER... %s", "OK")
		for {
			msg, ok := <-SMTPQueue
			defer close(SMTPQueue)
			if !ok {
				log.Println("The SMTP queue channel is closed, the processing of messages is stopped")
				return
			}
			log.Print("Sending EMail messages...")

			server := mail.NewSMTPClient()

			// SMTP Server
			server.Host = AppConfig.SMTP.HOST
			server.Port = AppConfig.SMTP.PORT
			server.Username = AppConfig.SMTP.USERNAME
			server.Password = AppConfig.SMTP.PASSWORD
			if AppConfig.SMTP.TLS {
				server.Encryption = mail.EncryptionTLS
			} else {
				server.Encryption = mail.EncryptionNone
			}

			// You can specified authentication type:
			// - AUTO (default)
			// - PLAIN
			// - LOGIN
			// - CRAM-MD5
			// - None
			switch AppConfig.SMTP.AUTH {
			case "auto":
				server.Authentication = mail.AuthAuto
			case "plain":
				server.Authentication = mail.AuthPlain
			case "login":
				server.Authentication = mail.AuthLogin
			case "crammd5", "cram-md5":
				server.Authentication = mail.AuthCRAMMD5
			case "none":
				server.Authentication = mail.AuthNone
			default:
				server.Authentication = mail.AuthAuto
			}

			// Variable to keep alive connection
			server.KeepAlive = AppConfig.SMTP.KEEPALIVE

			// Timeout for connect to SMTP Server
			server.ConnectTimeout = time.Duration(AppConfig.SMTP.TIMEOUTCONNECT) * time.Second

			// Timeout for send the data and wait respond
			server.SendTimeout = time.Duration(AppConfig.SMTP.TIMEOUTSEND) * time.Second

			if AppConfig.SMTP.TLS {
				server.TLSConfig = &tls.Config{
					InsecureSkipVerify: AppConfig.SMTP.SKIPVERIFY,
					MinVersion:         tls.VersionTLS12,
				}
			}

			for _, to := range msg.ToA {
				email := mail.NewMSG()
				email.SetFrom(AppConfig.SMTP.FROM).
					AddTo(to).
					SetSubject(msg.Subject)

				email.SetBody(mail.TextHTML, msg.Body)

				if email.Error != nil {
					log.Print(email.Error)
				}

				smtpClient, err := server.Connect()
				if err != nil {
					log.Print("Failed to connect to the SMTP server ", server.Host, ", ", server.Authentication, " (", err, ")")
				} else {
					// Call Send and pass the client
					err = email.Send(smtpClient)
					if err != nil {
						log.Print("EMail message failed to ", to, " (", err, ")")
					} else {
						log.Print("EMail message successfully sent to ", to)
					}
				}
			}
		}
	} else {
		log.Print("Sending EMail notification is not included")
	}
}
