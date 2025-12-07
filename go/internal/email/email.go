package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
)

type Sender struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func NewSender(host, port, username, password, from string) *Sender {
	return &Sender{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
	}
}

const verificationTemplate = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 5px; }
        .header { background-color: #6200ee; color: white; padding: 10px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { padding: 20px; }
        .button { display: inline-block; padding: 10px 20px; background-color: #03dac6; color: black; text-decoration: none; border-radius: 4px; font-weight: bold; }
        .footer { margin-top: 20px; font-size: 0.8em; color: #777; text-align: center; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to Chatty!</h1>
        </div>
        <div class="content">
            <p>Hi {{.Username}},</p>
            <p>Thanks for signing up for Chatty. Please verify your email address to get started.</p>
            <p style="text-align: center;">
                <a href="{{.Link}}" class="button">Verify Email</a>
            </p>
            <p>If you didn't create an account, you can safely ignore this email.</p>
        </div>
        <div class="footer">
            <p>&copy; 2025 Chatty Inc.</p>
        </div>
    </div>
</body>
</html>
`

func (s *Sender) SendVerificationEmail(to, username, link string) error {
	t, err := template.New("verification").Parse(verificationTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var body bytes.Buffer
	if err := t.Execute(&body, map[string]string{"Username": username, "Link": link}); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Email headers
	headers := make(map[string]string)
	headers["From"] = s.From
	headers["To"] = to
	headers["Subject"] = "Verify your Chatty email"
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"UTF-8\""

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body.String()

	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
	addr := fmt.Sprintf("%s:%s", s.Host, s.Port)

	// If no host is configured, just log it (for development/demo purposes if flags aren't set)
	if s.Host == "" {
		fmt.Println("==================================================")
		fmt.Printf("MOCK EMAIL TO: %s\n", to)
		fmt.Printf("SUBJECT: %s\n", headers["Subject"])
		fmt.Println(body.String())
		fmt.Println("==================================================")
		return nil
	}

	return smtp.SendMail(addr, auth, s.From, []string{to}, []byte(message))
}
