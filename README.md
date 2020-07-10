# mailbonobo

mailbonobo is a simple service which uses a preformatted gohtml template to generate inline html emails.

# How to Use

## Update Values

- in `/values`, open values.yml and change the values for each key/question. These values will later be injected into the gohtml template

## Mail Delivery Methods

### ad-hoc method
use this to send out emails manually from your local machine (tested with MacOS only)

#### prerequisite
open MacOS's native email application, then sign in using your account (e.g. Gmail, Outlook, etc.). You don't need to use this application for any other MacOS applications (e.g. calendar)

#### steps
1. open your terminal and navigate to this repo's directory
1. run `make gen-file` to generate a local html file, `output/<rfc number>.html`
1. open html file using Safari, then click File -> Share -> Email This Page to open a new email. The color scheme of the mail may look funny in the application, but it will look totoally normal to the recipients (I promise!)
1. delete the first line (`file:///.../mailbonobo/output/<rfc number>.html`)
1. check to make sure everything looks correct. Send a test email to yourself or some others to review
1. repeat step 3-4, add your recipients and make any final changes, then send out the email!

### using AWS SES

<mark>SES method is in alpha phase. The specific issue is that some servers (e.g. Outlook Exchange) may be marking emails from SES as spam and blocking them, thus preventing your emails from getting delivered to your beloved recipients :(</mark>

- in an AWS account, add your own email address as a verified sender email in SES. Please only use your email address, as AWS will send a verification email to the email address you just added
- run `go run main.go --valuesFile=values/YOUR_FILE_NAME` to send out an email to the recipient from the sender. Optionally, you may use the following flags:
    - awsRegion (default: us-east-1)
    - awsAccount (default: nonprod)
    - configFile (default: values/values.yaml)

#### Clean Up
remove your email address from SES. This step is optional but it is safer since bad actors can't use your email address to send emails to others...