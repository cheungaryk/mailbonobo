package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	log "github.com/sirupsen/logrus"

	//go get -u github.com/aws/aws-sdk-go
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"gopkg.in/yaml.v2"

	flag "github.com/spf13/pflag"
)

const (
	charSet      = "UTF-8"
	templateFile = "templates/main.gohtml"

	// Specify a configuration set. To use a configuration
	// set, comment the next line and line 92.
	//ConfigurationSet = "ConfigSet"
)

// content contains variables to be injected into the html template
// it reads the values from values.yaml for now
type mail struct {
	Metadata metadata `yaml:"metadata"` // sender and recipient info

	Title   string `yaml:"title"`   // 1-line description of the RFC
	Summary string `yaml:"summary"` // summary of the RFC. Brief but longer than title

	Ticket  ticket  `yaml:"ticket"`  // RFC and Jira ticket numbers
	Service service `yaml:"service"` // name and description of the service being operated on
	Content content `yaml:"content"` // more detail on the RFC (e.g. when, why, impact)
}

type metadata struct {
	Sender        string   `yaml:"sender"`        // Sender email address
	bccRecipients []string `yaml:"bccRecipients"` // Recipient email addresses
	SlackChannel  string   `yaml:"slackChannel"`  // Slack channel without the pound (e.g. gitlab)
}

type ticket struct {
	RFC        string `yaml:"rfc"`                 // RFC number (e.g. RFC-1234)
	JiraTicket string `yaml:"jiraTicket"`          // Jira ticket number (e.g. TOSD-123)
	RFCSubstr  string `yaml:"rfcSubstr,omitempty"` // used for RFC URL, do not specify in values.yaml
}

type service struct {
	ServiceName        string `yaml:"serviceName"`        // name of the service
	ServiceDescription string `yaml:"serviceDescription"` // describe what the service is or does
}

type content struct {
	When   string `yaml:"when"`               // maintenance date and time (e.g. 5 - 5:30 PM PT, Thursday, Apr 23th, 2020)
	Who    string `yaml:"whoIsBeingImpacted"` // who does it impact?
	Why    string `yaml:"why"`                // why are we doing the rfc?
	Outage string `yaml:"outage"`             // is there any outage?
}

func assembleEmail(sender string, bccRecipients []string, htmlBody string, subject string) *ses.SendEmailInput {
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			BccAddresses: aws.StringSlice(bccRecipients),
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(charSet),
					Data:    aws.String(htmlBody),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(charSet),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(sender),
	}
	return input
}

// parseHTMLBody reads the main.gohtml template and injects the variables, and returns it as a string so it can be sent out as an HTML body
func (m *mail) parseHTMLBody(tFile string) (string, error) {
	// read mail.gohtml as a template
	var b bytes.Buffer
	t, err := template.ParseFiles(tFile)
	if err != nil {
		log.WithFields(log.Fields{
			"file name": tFile,
		}).Errorf("[parseHTMLBody] cannot read file")
		return "", err
	}

	// inject the values into the template
	err = t.Execute(&b, m)
	if err != nil {
		log.WithFields(log.Fields{
			"file name": tFile,
		}).Errorf("[parseHTMLBody] evaluation error")
		return "", err
	}

	return b.String(), nil
}

// readYaml takes a file name as paramter, and unmarshal the file content into a *content object, then return the object
func (m *mail) readYaml(yamlFile string) (*mail, error) {
	// read the yaml file. Errors out if file does not exist
	y, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		log.WithFields(log.Fields{
			"file name": yamlFile,
		}).Errorf("[readYaml] cannot read file")
		return m, err
	}

	// unmarshal the file yaml content into the *mail struct
	err = yaml.Unmarshal(y, &m)
	if err != nil {
		log.WithFields(log.Fields{
			"file name": yamlFile,
		}).Errorf("[readYaml] unmarshal error")
		return m, err
	}

	return m, nil
}

// saveFile writes htmlBody as the content into a file named $fileName.
// if the file named <fileName> exists, its content will be overwritten.
// otherwise, the file will be created
func (m *mail) saveFile(fileName string, htmlBody string) error {
	f, err := os.Create(fileName) // be careful! If this file already exists, its content will be rewritten
	if err != nil {
		log.WithFields(log.Fields{
			"file name": fileName,
		}).Errorf("[saveFile] failed creating file")
		return err
	}

	defer f.Close() // Make sure to close the file when you're done

	// write the htmlBody into the file
	_, err = f.WriteString(htmlBody)
	if err != nil {
		log.WithFields(log.Fields{
			"file name": fileName,
		}).Errorf("[saveFile] failed writing to file")
		return err
	}

	log.WithFields(log.Fields{
		"file name": f.Name(),
	}).Infof("[saveFile] file created. See readme for sending it out (ad-hoc method)")
	return nil
}

func (m *mail) sendEmail(awsAccount string, awsRegion string, subject string, htmlBody string) (*ses.SendEmailOutput, error) {
	svc, err := newSESSession(awsAccount, awsRegion)
	if err != nil {
		return nil, err
	}

	input := assembleEmail(m.Metadata.Sender, m.Metadata.bccRecipients, htmlBody, subject)

	// attempt to send the email
	seo, err := svc.SendEmail(input)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			log.WithFields(log.Fields{
				"error code": aerr.Code(),
			}).Errorf("[sendEmail] AWS error - message rejected")
			return seo, aerr
		} else {
			log.WithFields(log.Fields{
				"error code": err,
			}).Errorf("[sendEmail] non-AWS error - message rejected")
			return seo, err
		}
	}

	return seo, nil
}

func newSESSession(awsAccount string, awsRegion string) (*ses.SES, error) {
	// create a new AWS session
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewSharedCredentials("", awsAccount),
		Region:      aws.String(awsRegion)},
	)
	if err != nil {
		log.Errorf("[sesSession] failed to create AWS session")
		return nil, err
	}

	// create an SES session
	return ses.New(sess), nil
}

func main() {
	var awsRegion string
	flag.StringVar(&awsRegion, "awsRegion", "us-east-1", "AWS region (default: us-east-1)")

	var awsAccount string
	flag.StringVar(&awsAccount, "awsAccount", "tm-nonprod-Ops-Techops", "AWS Account (default: nonprod)")

	var valuesFile string
	flag.StringVar(&valuesFile, "valuesFile", "values/values.yml", "values yaml file (default: values/values.yml)")

	var saveFile bool
	flag.BoolVar(&saveFile, "saveFile", false, "save file instead of sending out email (default: false)")

	flag.Parse()

	// Read the values from the yaml
	var m *mail
	m, err := m.readYaml(valuesFile)
	if err != nil {
		log.Fatalf(err.Error())
	}
	m.Ticket.RFCSubstr = m.Ticket.RFC[4:] // this variable is for constructing the RFC URL

	// Construct subject and html body
	subject := fmt.Sprintf("%s - %s", m.Ticket.RFC, m.Title)
	htmlBody, err := m.parseHTMLBody(templateFile)
	if err != nil {
		log.Fatalf(err.Error())
	}

	// if saveFile is used, instead of sending an email, it will create a template html file to be sent out instead
	if saveFile {
		err = m.saveFile(fmt.Sprintf("output/%s.html", m.Ticket.RFC), htmlBody)
		if err != nil {
			log.Fatalf(err.Error())
		}
	} else {
		result, err := m.sendEmail(awsAccount, awsRegion, subject, htmlBody)

		// Display error messages if they occur.
		if err != nil {
			return
		}

		log.Infof("Email sent to addresses: %s", m.Metadata.bccRecipients)
		log.Debugf(result.String())
	}

}
