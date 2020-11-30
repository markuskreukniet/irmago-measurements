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

	KssGetCommitments    int = iota
	KssGetProofPs        int = iota
	TorKssGetCommitments int = iota
	TorKssGetProofPs     int = iota

	DisclosureHttpsNewSession           int = iota
	DisclosureHttpsRespondPermission    int = iota
	IssuanceHttpsNewSession             int = iota
	IssuanceHttpsRespondPermission      int = iota
	TorDisclosureHttpsNewSession        int = iota
	TorDisclosureHttpsRespondPermission int = iota
	TorIssuanceHttpsNewSession          int = iota
	TorIssuanceHttpsRespondPermission   int = iota

	KssHttpsGetCommitments    int = iota
	KssHttpsGetProofPs        int = iota
	TorKssHttpsGetCommitments int = iota
	TorKssHttpsGetProofPs     int = iota

	measurementsDoneText string = "measurements done: "

	folderPath      string = "/data/user/0/foundation.privacybydesign.irmamobile.alpha/v2"
	filePart        string = "/measurementsDone.txt"
	filePartFlutter string = "/latestMeasurementsFlutter.txt"
	filePath        string = folderPath + filePart
	filePathFlutter string = folderPath + filePartFlutter

	filePartDisclosureNewSession           string = "/disclosureNewSession.txt"
	filePartDisclosureRespondPermission    string = "/disclosureRespondPermission.txt"
	filePartIssuanceNewSession             string = "/issuanceNewSession.txt"
	filePartIssuanceRespondPermission      string = "/issuanceRespondPermission.txt"
	filePartTorDisclosureNewSession        string = "/torDisclosureNewSession.txt"
	filePartTorDisclosureRespondPermission string = "/torDisclosureRespondPermission.txt"
	filePartTorIssuanceNewSession          string = "/torIssuanceNewSession.txt"
	filePartTorIssuanceRespondPermission   string = "/torIssuanceRespondPermission.txt"

	filePartKssGetCommitments    string = "/kssGetCommitments.txt"
	filePartKssGetProofPs        string = "/kssGetProofPs.txt"
	filePartTorKssGetCommitments string = "/torKssGetCommitments.txt"
	filePartTorKssGetProofPs     string = "/torKssGetProofPs.txt"

	// HTTPS - start
	filePartDisclosureHttpsNewSession           string = "/disclosureHttpsNewSession.txt"
	filePartDisclosureHttpsRespondPermission    string = "/disclosureHttpsRespondPermission.txt"
	filePartIssuanceHttpsNewSession             string = "/issuanceHttpsNewSession.txt"
	filePartIssuanceHttpsRespondPermission      string = "/issuanceHttpsRespondPermission.txt"
	filePartTorDisclosureHttpsNewSession        string = "/torDisclosureHttpsNewSession.txt"
	filePartTorDisclosureHttpsRespondPermission string = "/torDisclosureHttpsRespondPermission.txt"
	filePartTorIssuanceHttpsNewSession          string = "/torIssuanceHttpsNewSession.txt"
	filePartTorIssuanceHttpsRespondPermission   string = "/torIssuanceHttpsRespondPermission.txt"

	filePartKssHttpsGetCommitments    string = "/kssHttpsGetCommitments.txt"
	filePartKssHttpsGetProofPs        string = "/kssHttpsGetProofPs.txt"
	filePartTorKssHttpsGetCommitments string = "/torKssHttpsGetCommitments.txt"
	filePartTorKssHttpsGetProofPs     string = "/torKssHttpsGetProofPs.txt"
	// HTTPS - end

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

func determineFlutterMeasurementText(measurementType int) string {
	flutterMeasurementText := ""

	switch measurementType {
	case DisclosureNewSession:
		flutterMeasurementText = "disclosureNewSession: "
	case DisclosureRespondPermission:
		flutterMeasurementText = "\ndisclosureRespondPermission: "
	case IssuanceNewSession:
		flutterMeasurementText = "issuanceNewSession: "
	case IssuanceRespondPermission:
		flutterMeasurementText = "\nissuanceRespondPermission: "
	case TorDisclosureNewSession:
		flutterMeasurementText = "torDisclosureNewSession: "
	case TorDisclosureRespondPermission:
		flutterMeasurementText = "\ntorDisclosureRespondPermission: "
	case TorIssuanceNewSession:
		flutterMeasurementText = "torIssuanceNewSession: "
	case TorIssuanceRespondPermission:
		flutterMeasurementText = "\ntorIssuanceRespondPermission: "

	case KssGetCommitments:
		flutterMeasurementText = "\nkssGetCommitments: "
	case KssGetProofPs:
		flutterMeasurementText = "\nkssGetProofPs: "
	case TorKssGetCommitments:
		flutterMeasurementText = "\ntorKssGetCommitments: "
	case TorKssGetProofPs:
		flutterMeasurementText = "\ntorKssGetProofPs: "

		// HTTPS - start
	case DisclosureHttpsNewSession:
		flutterMeasurementText = "disclosureHttpsNewSession: "
	case DisclosureHttpsRespondPermission:
		flutterMeasurementText = "\ndisclosureHttpsRespondPermission: "
	case IssuanceHttpsNewSession:
		flutterMeasurementText = "issuanceHttpsNewSession: "
	case IssuanceHttpsRespondPermission:
		flutterMeasurementText = "\nissuanceHttpsRespondPermission: "
	case TorDisclosureHttpsNewSession:
		flutterMeasurementText = "torDisclosureHttpsNewSession: "
	case TorDisclosureHttpsRespondPermission:
		flutterMeasurementText = "\ntorDisclosureHttpsRespondPermission: "
	case TorIssuanceHttpsNewSession:
		flutterMeasurementText = "torIssuanceHttpsNewSession: "
	case TorIssuanceHttpsRespondPermission:
		flutterMeasurementText = "\ntorIssuanceHttpsRespondPermission: "

	case KssHttpsGetCommitments:
		flutterMeasurementText = "\nkssHttpsGetCommitments: "
	case KssHttpsGetProofPs:
		flutterMeasurementText = "\nkssHttpsGetProofPs: "
	case TorKssHttpsGetCommitments:
		flutterMeasurementText = "\ntorKssHttpsGetCommitments: "
	case TorKssHttpsGetProofPs:
		flutterMeasurementText = "\ntorKssHttpsGetProofPs: "
		// HTTPS - end
	}

	return flutterMeasurementText
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

func SendResultsAndResetMeasurements() {
	var filePaths []string
	emailText := ""

	disclosureNewSessionAverageFilePath := folderPath +
		filePartDisclosureNewSession

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		disclosureNewSessionAverageFilePath,
		emailText,
		"disclosure new session")

	disclosureRespondPermissionAverageFilePath := folderPath +
		filePartDisclosureRespondPermission

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		disclosureRespondPermissionAverageFilePath,
		emailText,
		"disclosure respond permission")

	issuanceNewSessionAverageFilePath := folderPath +
		filePartIssuanceNewSession

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		issuanceNewSessionAverageFilePath,
		emailText,
		"issuance new session")

	issuanceRespondPermissionAverageFilePath := folderPath +
		filePartIssuanceRespondPermission

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		issuanceRespondPermissionAverageFilePath,
		emailText,
		"issuance respond permission")

	torDisclosureNewSessionAverageFilePath := folderPath +
		filePartTorDisclosureNewSession

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torDisclosureNewSessionAverageFilePath,
		emailText,
		"disclosure new session over Tor")

	torDisclosureRespondPermissionAverageFilePath := folderPath +
		filePartTorDisclosureRespondPermission

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torDisclosureRespondPermissionAverageFilePath,
		emailText,
		"disclosure respond permission over Tor")

	torIssuanceNewSessionAverageFilePath := folderPath +
		filePartTorIssuanceNewSession

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torIssuanceNewSessionAverageFilePath,
		emailText,
		"issuance new session over Tor")

	torIssuanceRespondPermissionAverageFilePath := folderPath +
		filePartTorIssuanceRespondPermission

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torIssuanceRespondPermissionAverageFilePath,
		emailText,
		"issuance respond permission over Tor")

	// KSS - start
	kssGetCommitmentsAverageFilePath := folderPath +
		filePartKssGetCommitments

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		kssGetCommitmentsAverageFilePath,
		emailText,
		"KSS GetCommitments")

	kssGetProofPsAverageFilePath := folderPath +
		filePartKssGetProofPs

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		kssGetProofPsAverageFilePath,
		emailText,
		"KSS GetProofPs")

	torKssGetCommitmentsAverageFilePath := folderPath +
		filePartTorKssGetCommitments

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torKssGetCommitmentsAverageFilePath,
		emailText,
		"Tor KSS GetCommitments")

	torKssGetProofPsAverageFilePath := folderPath +
		filePartTorKssGetProofPs

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torKssGetProofPsAverageFilePath,
		emailText,
		"Tor KSS GetProofPs")
	// KSS - end

	// HTTPS - start
	disclosureHttpsNewSessionAverageFilePath := folderPath +
		filePartDisclosureHttpsNewSession

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		disclosureHttpsNewSessionAverageFilePath,
		emailText,
		"disclosure new session over HTTPS")

	disclosureHttpsRespondPermissionAverageFilePath := folderPath +
		filePartDisclosureHttpsRespondPermission

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		disclosureHttpsRespondPermissionAverageFilePath,
		emailText,
		"disclosure respond permission over HTTPS")

	issuanceHttpsNewSessionAverageFilePath := folderPath +
		filePartIssuanceHttpsNewSession

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		issuanceHttpsNewSessionAverageFilePath,
		emailText,
		"issuance new session over HTTPS")

	issuanceHttpsRespondPermissionAverageFilePath := folderPath +
		filePartIssuanceHttpsRespondPermission

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		issuanceHttpsRespondPermissionAverageFilePath,
		emailText,
		"issuance respond permission over HTTPS")

	torDisclosureHttpsNewSessionAverageFilePath := folderPath +
		filePartTorDisclosureHttpsNewSession

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torDisclosureHttpsNewSessionAverageFilePath,
		emailText,
		"disclosure new session over Tor and HTTPS")

	torDisclosureHttpsRespondPermissionAverageFilePath := folderPath +
		filePartTorDisclosureHttpsRespondPermission

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torDisclosureHttpsRespondPermissionAverageFilePath,
		emailText,
		"disclosure respond permission over Tor and HTTPS")

	torIssuanceHttpsNewSessionAverageFilePath := folderPath +
		filePartTorIssuanceHttpsNewSession

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torIssuanceHttpsNewSessionAverageFilePath,
		emailText,
		"issuance new session over Tor and HTTPS")

	torIssuanceHttpsRespondPermissionAverageFilePath := folderPath +
		filePartTorIssuanceHttpsRespondPermission

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torIssuanceHttpsRespondPermissionAverageFilePath,
		emailText,
		"issuance respond permission over Tor and HTTPS")

	// KSS - start
	kssHttpsGetCommitmentsAverageFilePath := folderPath +
		filePartKssHttpsGetCommitments

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		kssHttpsGetCommitmentsAverageFilePath,
		emailText,
		"KSS GetCommitments over HTTPS")

	kssHttpsGetProofPsAverageFilePath := folderPath +
		filePartKssHttpsGetProofPs

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		kssHttpsGetProofPsAverageFilePath,
		emailText,
		"KSS GetProofPs over HTTPS")

	torKssHttpsGetCommitmentsAverageFilePath := folderPath +
		filePartTorKssHttpsGetCommitments

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torKssHttpsGetCommitmentsAverageFilePath,
		emailText,
		"Tor KSS GetCommitments over HTTPS")

	torKssHttpsGetProofPsAverageFilePath := folderPath +
		filePartTorKssHttpsGetProofPs

	filePaths, emailText = addFilePathAndEmailTextIfExist(filePaths,
		torKssHttpsGetProofPsAverageFilePath,
		emailText,
		"Tor KSS GetProofPs over HTTPS")
	// KSS - end
	// HTTPS - end

	emailText += "The averages are in microseconds."

	sendMail(emailText, filePaths)

	for _, filePath := range filePaths {
		deleteMeasurementsDoneFile(filePath)
	}

	replaceFileContentWithString(filePath, measurementsDoneText+"0")
}

func AddMeasurementResult(measurementType int, result int64) {
	filePath := ""

	switch measurementType {
	case DisclosureNewSession:
		filePath = folderPath + filePartDisclosureNewSession
	case DisclosureRespondPermission:
		filePath = folderPath + filePartDisclosureRespondPermission
	case IssuanceNewSession:
		filePath = folderPath + filePartIssuanceNewSession
	case IssuanceRespondPermission:
		filePath = folderPath + filePartIssuanceRespondPermission
	case TorDisclosureNewSession:
		filePath = folderPath + filePartTorDisclosureNewSession
	case TorDisclosureRespondPermission:
		filePath = folderPath + filePartTorDisclosureRespondPermission
	case TorIssuanceNewSession:
		filePath = folderPath + filePartTorIssuanceNewSession
	case TorIssuanceRespondPermission:
		filePath = folderPath + filePartTorIssuanceRespondPermission

	case KssGetCommitments:
		filePath = folderPath + filePartKssGetCommitments
	case KssGetProofPs:
		filePath = folderPath + filePartKssGetProofPs
	case TorKssGetCommitments:
		filePath = folderPath + filePartTorKssGetCommitments
	case TorKssGetProofPs:
		filePath = folderPath + filePartTorKssGetProofPs

		// HTTPS - start
	case DisclosureHttpsNewSession:
		filePath = folderPath + filePartDisclosureHttpsNewSession
	case DisclosureHttpsRespondPermission:
		filePath = folderPath + filePartDisclosureHttpsRespondPermission
	case IssuanceHttpsNewSession:
		filePath = folderPath + filePartIssuanceHttpsNewSession
	case IssuanceHttpsRespondPermission:
		filePath = folderPath + filePartIssuanceHttpsRespondPermission
	case TorDisclosureHttpsNewSession:
		filePath = folderPath + filePartTorDisclosureHttpsNewSession
	case TorDisclosureHttpsRespondPermission:
		filePath = folderPath + filePartTorDisclosureHttpsRespondPermission
	case TorIssuanceHttpsNewSession:
		filePath = folderPath + filePartTorIssuanceHttpsNewSession
	case TorIssuanceHttpsRespondPermission:
		filePath = folderPath + filePartTorIssuanceHttpsRespondPermission

	case KssHttpsGetCommitments:
		filePath = folderPath + filePartKssHttpsGetCommitments
	case KssHttpsGetProofPs:
		filePath = folderPath + filePartKssHttpsGetProofPs
	case TorKssHttpsGetCommitments:
		filePath = folderPath + filePartTorKssHttpsGetCommitments
	case TorKssHttpsGetProofPs:
		filePath = folderPath + filePartTorKssHttpsGetProofPs
		// HTTPS - end
	}

	stringContent := ""

	if pathDoesExist(filePath) {
		stringContent = determineFileStringContent(filePath)
	}

	possibleLineFeed := "\n"
	if stringContent == "" {
		possibleLineFeed = ""
	}

	stringResult := strconv.FormatInt(result, 10)

	stringContent += possibleLineFeed + measurementText + stringResult

	replaceFileContentWithString(filePath, stringContent)

	// for Flutter part
	stringContentFlutter := ""

	if pathDoesExist(filePathFlutter) {
		stringContentFlutter = determineFileStringContent(filePathFlutter)
	}

	stringContentFlutter +=
		determineFlutterMeasurementText(measurementType) + stringResult

	replaceFileContentWithString(filePathFlutter, stringContentFlutter)
}

func StopProgramWhenNeeded(useTor bool, httpClient *http.Client) {
	if httpClient != nil {
		if useTor && !IsClientConnectedToTor(httpClient) {
			log.Fatal("use Tor && client is not connected to Tor")
		}
	}
}

func ClearFlutterMeasurements() {
	replaceFileContentWithString(filePathFlutter, "")
}
