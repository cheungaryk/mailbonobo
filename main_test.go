package main

import (
	"os"
	"strings"
	"testing"
)

func TestReadYaml(t *testing.T) {
	tests := []struct {
		file   string
		errMsg string
	}{
		{
			file:   "",
			errMsg: "no such file or directory",
		},
		{
			file:   "doesNotExist.yml",
			errMsg: "no such file or directory",
		},
		{
			file:   "testfiles/bad_values.yml",
			errMsg: "yaml: unmarshal errors",
		},
		{
			file:   "values/values.yml",
			errMsg: "",
		},
	}

	for i, test := range tests {
		var m *mail
		m, err := m.readYaml(test.file)
		if err != nil && !strings.Contains(err.Error(), test.errMsg) {
			t.Errorf("test case %d: expected error: %v, got unexpected error: %v", i+1, test.errMsg, err)
		}
	}
}

func TestParseHTMLBody(t *testing.T) {
	tests := []struct {
		file   string
		errMsg string
	}{
		{
			file:   "",
			errMsg: "no such file or directory",
		},
		{
			file:   "doesNotExist.gohtml",
			errMsg: "no such file or directory",
		},
		{
			file:   "testfiles/bad.gohtml",
			errMsg: "can't evaluate field",
		},
		{
			file:   "templates/main.gohtml",
			errMsg: "",
		},
	}

	for i, test := range tests {
		var m *mail
		m, _ = m.readYaml("values/values.yml")
		output, err := m.parseHTMLBody(test.file)
		if err != nil {
			if !strings.Contains(err.Error(), test.errMsg) {
				t.Errorf("test case %d: expected error: %v, got unexpected error: %v", i+1, test.errMsg, err)
			}
		} else if !strings.HasPrefix(output, "<!DOCTYPE html") {
			t.Errorf("test case %d: output is not valid html: %v", i+1, output)
		}
	}
}

func TestSaveFile(t *testing.T) {
	tests := []struct {
		fileName string
		htmlBody string
	}{
		{
			fileName: "testfiles/testSaveFile.yaml",
			htmlBody: "testing testing",
		},
	}

	for i, test := range tests {
		var m *mail
		m.saveFile(test.fileName, test.htmlBody)

		if _, err := os.Stat(test.fileName); err != nil {
			if os.IsNotExist(err) {
				t.Errorf("test case %d: file did not get created: %s", i, test.fileName)
			}
		}

		os.Remove(test.fileName)
	}
}

func TestNewSESSession(t *testing.T) {
	tests := []struct {
		awsAccount string
		awsRegion  string
	}{
		{
			awsAccount: "fakeAccount",
			awsRegion:  "us-east-1",
		},
	}

	for i, test := range tests {
		_, err := newSESSession(test.awsAccount, test.awsRegion)

		if err != nil {
			t.Errorf("test case %d: got unexpected error: %v", i+1, err)
		}
	}
}

func TestAssembleEmail(t *testing.T) {
	tests := []struct {
		sender        string
		bccRecipients []string
		htmlBody      string
		subject       string
	}{
		{
			sender:        "a@company.com",
			bccRecipients: []string{"a", "b"},
			htmlBody:      "abc",
			subject:       "subject",
		},
	}

	for i, test := range tests {
		input := assembleEmail(test.sender, test.bccRecipients, test.htmlBody, test.subject)

		if *input.Source != test.sender {
			t.Errorf("test case %v: expected sender as %v, got %v", i, test.sender, *input.Source)
		}

		if *input.Message.Body.Html.Data != test.htmlBody {
			t.Errorf("test case %v: expected sender as %v, got %v", i, test.htmlBody, *input.Message.Body.Html.Data)
		}

		if *input.Message.Subject.Data != test.subject {
			t.Errorf("test case %v: expected sender as %v, got %v", i, test.subject, *input.Message.Subject.Data)
		}
	}
}
