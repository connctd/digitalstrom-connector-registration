package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/connctd/digitalstrom"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/google/uuid"
)

type accountRow struct {
	link    string
	user    string
	secret  string
	token   string
	success bool
	err     string
}

type AccountExport struct {
	Link      string
	Token     string
	SubjectId string
}

const (
	COL_NAMES          = "url;username;password"
	FILE_NAME          = "accounts.csv"
	FILE_NAME_EXPORT   = "tokens.json"
	FILE_NAME_PROTOCOL = "report.log"
	FILE_NAME_LOG      = "debug.log"
	APP_NAME           = "foresight-connctd" // the name visible for access rights in the dS system
)

var logger logr.Logger

func main() {
	// set the logger - writing everything to log.txt
	setLogger()
	// use the account object of the digitalstrom library, needed to register applications
	dSAccount := *digitalstrom.NewAccount()
	// fix for TLS issue, bug of digitalStrom
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	dSAccount.Connection.HTTPClient = client
	// set filename, check if given via argument
	filename := FILE_NAME
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}

	fmt.Println("")
	fmt.Printf("This program will register the CONNCTD connector to all dS-Accounts, given in the file %s.\n", filename)
	fmt.Println("")
	fmt.Printf("Reading file %s  ... ", filename)
	logger.Info(fmt.Sprintf("reading file %s", filename))
	//open the given file
	readFile, err := os.Open(filename)
	defer readFile.Close()
	if err != nil {
		logger.Error(err, fmt.Sprintf("file %s", filename))
		fmt.Println("ERROR")
		fmt.Println(err)
		fmt.Println("Either name your csv file accounts.csv and run the program without arguments or name the file by argument when calling the program.")
		fmt.Println("Program stopped")
		return
	}
	fmt.Println("OK")
	// use line scanner to read csv file line by line
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	// scan for first line
	fileScanner.Scan()
	firstLine := fileScanner.Text()
	logger.Info(fmt.Sprintf("firstline of file '%s' is '%s'", filename, firstLine))
	if firstLine != COL_NAMES {
		logger.Error(fmt.Errorf("first line does not match with expected line ('%s'))", COL_NAMES), "program aborted")
		fmt.Printf("wrong column names, must be %s\n", COL_NAMES)
		fmt.Println("Program stopped")
		return
	}
	// first line is ok, prepare line reading
	accounts := []*accountRow{}
	// scan all other lines

	for fileScanner.Scan() {
		newLine := fileScanner.Text()
		logger.Info(fmt.Sprintf("reading next line '%s'", newLine))
		newAccount, err := getAccountRow(newLine)
		if err != nil {
			logger.Error(err, "program aborted")
			fmt.Printf("error, unable to read line: '%s' (%s)\n", newLine, err)
			fmt.Println("Program stopped")
			return
		}
		accounts = append(accounts, newAccount)

	}

	fmt.Println("")
	fmt.Printf("Found %d rows with account data.\n", len(accounts))
	fmt.Println("")
	fmt.Println("Will now continue with connector registration:")

	for i := range accounts {
		fmt.Printf("   %d/%d ", i+1, len(accounts))
		registerConnector(&dSAccount, accounts[i])
	}

	fmt.Println("")
	fmt.Printf("Exporting links and appliation tokens to file '%s'  ....", FILE_NAME_EXPORT)
	err = saveAccountData(accounts, FILE_NAME_EXPORT)
	if err != nil {
		fmt.Printf("ERROR (%s)", err)
	} else {
		fmt.Println("OK")
	}

	fmt.Printf("Exporting success report to file '%s'  ..................", FILE_NAME_PROTOCOL)
	err = saveReport(accounts, FILE_NAME_PROTOCOL)
	if err != nil {
		fmt.Printf("ERROR (%s)", err)
	} else {
		fmt.Println("OK")
	}

	fmt.Printf("Program finished.\n")

	fmt.Println("")
	fmt.Println("")
}

func saveAccountData(accounts []*accountRow, filename string) error {
	logger.Info(fmt.Sprintf("saving export to file '%s'", filename))
	export := []AccountExport{}

	for i := range accounts {
		if accounts[i].success {
			entry := AccountExport{
				Link:      accounts[i].link,
				Token:     accounts[i].token,
				SubjectId: uuid.New().String(),
			}
			export = append(export, entry)
		}
	}

	res, err := json.Marshal(export)

	if err != nil {
		logger.Error(err, "failed to convert array of AccountExport to json")
		return err
	}

	if err := os.WriteFile(filename, res, 0666); err != nil {
		logger.Error(err, "unable to save json %s", res)
		return err
	}

	return nil

}

func saveReport(accounts []*accountRow, filename string) error {
	lines := []string{}
	logger.Info(fmt.Sprintf("Saving report to file '%s'", filename))
	for i := range accounts {
		if accounts[i].success {
			lines = append(lines, fmt.Sprintf("SUCCESS %s ", accounts[i].link))
		} else {
			lines = append(lines, fmt.Sprintf("FAIL    %s (%s)", accounts[i].link, accounts[i].err))
		}
	}
	f, err := os.Create(filename)
	if err != nil {
		logger.Error(err, "unable to save report %s", lines)
		return err
	}
	// remember to close the file
	defer f.Close()

	for _, line := range lines {
		_, err := fmt.Fprintln(f, line)
		if err != nil {
			logger.Error(err, "unable to save report %s", lines)
			return err
		}
	}

	return nil
}

func registerConnector(dSAccount *digitalstrom.Account, accountRow *accountRow) {
	fmt.Printf(" registering application '%s' at %s .... ", APP_NAME, accountRow.link)
	logger.Info(fmt.Sprintf(" registering application '%s' at %s .... ", APP_NAME, accountRow.link))
	dSAccount.SetURL(accountRow.link)

	atoken, err := dSAccount.RegisterApplication(APP_NAME, accountRow.user, accountRow.secret)
	if err != nil {
		logger.Error(err, "registration marked as failed for this account")
		fmt.Println("ERROR")
		accountRow.err = fmt.Sprintf("%s", err)
		return
	}
	fmt.Println("OK")
	accountRow.token = atoken
	accountRow.success = true
	accountRow.err = ""
}

func getAccountRow(line string) (*accountRow, error) {

	values := strings.Split(line, ";")
	if len(values) != 3 {
		return nil, fmt.Errorf("line does not contains of 3 columns (%s)", line)
	}
	newRow := accountRow{
		link:    values[0],
		user:    values[1],
		secret:  values[2],
		token:   "",
		success: false,
	}
	return &newRow, nil
}
func setLogger() {
	f, err := os.OpenFile(FILE_NAME_LOG, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	l := stdr.New(log.New(f, "", log.LstdFlags|log.Lshortfile))

	logger = l.WithName("ds-connector-registration")
	logger.Info("setting logger at digitalstrom library")
	digitalstrom.SetLogger(l)

}
