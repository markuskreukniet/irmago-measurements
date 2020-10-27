package irma

import (
	"bufio"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"

	"github.com/jordan-wright/email"
)

const (
	DisclosureNewSession           int = iota
	DisclosureRespondPermission    int = iota
	IssuanceNewSession             int = iota
	IssuanceRespondPermission      int = iota
	TorDisclosureNewSession        int = iota
	TorDisclosureRespondPermission int = iota
	TorIssuanceNewSession          int = iota
	TorIssuanceRespondPermission   int = iota

	measurementsDoneText string = "measurements done: "

	folderPath string = "/data/user/0/foundation.privacybydesign.irmamobile.alpha/v2"
	filePart   string = "/measurementsDone.txt"
	filePath   string = folderPath + filePart

	filePartDisclosureNewSession           string = "/disclosureNewSession.txt"
	filePartDisclosureRespondPermission    string = "/disclosureRespondPermission.txt"
	filePartIssuanceNewSession             string = "/issuanceNewSession.txt"
	filePartIssuanceRespondPermission      string = "/issuanceRespondPermission.txt"
	filePartTorDisclosureNewSession        string = "/torDisclosureNewSession.txt"
	filePartTorDisclosureRespondPermission string = "/torDisclosureRespondPermission.txt"
	filePartTorIssuanceNewSession          string = "/torIssuanceNewSession.txt"
	filePartTorIssuanceRespondPermission   string = "/torIssuanceRespondPermission.txt"

	measurementText string = "measurement: "
)

// private functions

func pathDoesExist(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func replaceFileContentWithString(filePath string, s string) {
	bytes := []byte(s)
	err := ioutil.WriteFile(filePath, bytes, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func determineFileStringContent(filePath string) string {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}

	return string(content)
}

func determineMeasurementsDone(filePath string) int {
	if !pathDoesExist(filePath) {
		return 0
	}

	stringContent := determineFileStringContent(filePath)

	if strings.Contains(stringContent, measurementsDoneText) {
		stringNumber := strings.Split(stringContent,
			measurementsDoneText)[1]
		number, err := strconv.Atoi(stringNumber)
		if err != nil {
			log.Fatal(err)
		}

		return number
	} else {
		return 0
	}
}

func deleteMeasurementsDoneFile(filePath string) {
	if pathDoesExist(filePath) {
		err := os.Remove(filePath)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func calculateAverage(filePath string) int64 {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}

	var measurements []int64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), measurementText) {
			var s = strings.Split(scanner.Text(), " ")[1]

			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				log.Fatal(err)
			}

			measurements = append(measurements, i)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	if len(measurements) > 0 {
		var sum int64 = 0

		for _, i := range measurements {
			sum += i
		}

		return sum / int64(len(measurements))
	} else {
		return 0
	}
}

func sendMail(emailText string, filePaths []string) {
	smtpServerHost := "smtp.gmail.com"
	smtpServerAddress := smtpServerHost + ":587"
	emailAddress := "irmamobilemeasurementtests@gmail.com"

	e := email.NewEmail()

	// by testing, it looks like AttachFile has to happen first
	for _, filePath := range filePaths {
		e.AttachFile(filePath)
	}

	e.From = emailAddress
	e.To = []string{emailAddress}
	e.Subject = "measurement averages"
	e.Text = []byte(emailText)
	e.Send(smtpServerAddress, smtp.PlainAuth("", emailAddress, "asdf5asdf", smtpServerHost))
}

func addFilePathAndEmailTextIfExist(filePaths []string,
	filePath string,
	emailText string,
	measurement string) ([]string, string) {
	if pathDoesExist(filePath) {
		filePaths = append(filePaths, filePath)
		average := calculateAverage(filePath)
		emailText += "The " + measurement + " average is: " +
			strconv.FormatInt(average, 10) + "\n"
	}

	return filePaths, emailText
}

// public functions

func IncrementMeasurementAndDetermineAgain() bool {
	const totalMeasurements = 25

	measurementsDone := determineMeasurementsDone(filePath)
	measurementsDone++
	replaceFileContentWithString(
		filePath,
		measurementsDoneText+strconv.Itoa(measurementsDone),
	)

	if measurementsDone < totalMeasurements {
		return true
	} else {
		return false
	}
}

func SendResultsAndResetMeasurements(newFolderPaths ...string) {
	usableFolderPath := folderPath
	if len(newFolderPaths) > 0 {
		usableFolderPath = newFolderPaths[0]
	}

	var filePaths []string
	emailText := ""

	disclosureNewSessionAverageFilePath := usableFolderPath +
		filePartDisclosureNewSession

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		disclosureNewSessionAverageFilePath,
		emailText,
		"disclosure new session")

	disclosureRespondPermissionAverageFilePath := usableFolderPath +
		filePartDisclosureRespondPermission

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		disclosureRespondPermissionAverageFilePath,
		emailText,
		"disclosure respond permission")

	issuanceNewSessionAverageFilePath := usableFolderPath +
		filePartIssuanceNewSession

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		issuanceNewSessionAverageFilePath,
		emailText,
		"issuance new session")

	issuanceRespondPermissionAverageFilePath := usableFolderPath +
		filePartIssuanceRespondPermission

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		issuanceRespondPermissionAverageFilePath,
		emailText,
		"issuance respond permission")

	torDisclosureNewSessionAverageFilePath := usableFolderPath +
		filePartTorDisclosureNewSession

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torDisclosureNewSessionAverageFilePath,
		emailText,
		"disclosure new session over Tor")

	torDisclosureRespondPermissionAverageFilePath := usableFolderPath +
		filePartTorDisclosureRespondPermission

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torDisclosureRespondPermissionAverageFilePath,
		emailText,
		"disclosure respond permission over Tor")

	torIssuanceNewSessionAverageFilePath := usableFolderPath +
		filePartTorIssuanceNewSession

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torIssuanceNewSessionAverageFilePath,
		emailText,
		"issuance new session over Tor")

	torIssuanceRespondPermissionAverageFilePath := usableFolderPath +
		filePartTorIssuanceRespondPermission

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torIssuanceRespondPermissionAverageFilePath,
		emailText,
		"issuance respond permission over Tor")

	emailText += "The averages are in microseconds."

	sendMail(emailText, filePaths)

	for _, filePath := range filePaths {
		deleteMeasurementsDoneFile(filePath)
	}

	replaceFileContentWithString(filePath, measurementsDoneText+"0")
}

func AddMeasurementResult(measurementType int, result int64, newFolderPaths ...string) {
	usableFolderPath := folderPath
	if len(newFolderPaths) > 0 {
		usableFolderPath = newFolderPaths[0]
	}

	filePath := ""

	switch measurementType {
	case DisclosureNewSession:
		filePath = usableFolderPath + filePartDisclosureNewSession
	case DisclosureRespondPermission:
		filePath = usableFolderPath + filePartDisclosureRespondPermission
	case IssuanceNewSession:
		filePath = usableFolderPath + filePartIssuanceNewSession
	case IssuanceRespondPermission:
		filePath = usableFolderPath + filePartIssuanceRespondPermission
	case TorDisclosureNewSession:
		filePath = usableFolderPath + filePartTorDisclosureNewSession
	case TorDisclosureRespondPermission:
		filePath = usableFolderPath + filePartTorDisclosureRespondPermission
	case TorIssuanceNewSession:
		filePath = usableFolderPath + filePartTorIssuanceNewSession
	case TorIssuanceRespondPermission:
		filePath = usableFolderPath + filePartTorIssuanceRespondPermission
	}

	stringContent := ""

	if pathDoesExist(filePath) {
		stringContent = determineFileStringContent(filePath)
	}

	possibleLineFeed := "\n"
	if stringContent == "" {
		possibleLineFeed = ""
	}

	stringContent += possibleLineFeed + measurementText + strconv.FormatInt(result, 10)

	replaceFileContentWithString(filePath, stringContent)
}

func StopProgramWhenNeeded(useTor bool, httpClient *http.Client) {
	if httpClient != nil {
		if useTor && !IsClientConnectedToTor(httpClient) {
			log.Fatal("use Tor && client is not connected to Tor")
		}
	}
}
