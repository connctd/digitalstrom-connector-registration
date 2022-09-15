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
	COL_NAMES          = "link;user;secret"
	FILE_NAME          = "accounts.csv"
	FILE_NAME_EXPORT   = "tokens.json"
	FILE_NAME_PROTOCOL = "report.log"
	APP_NAME           = "foresight-connctd"
)

func main() {

	setLogger()
	dSAccount := *digitalstrom.NewAccount()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	dSAccount.Connection.HTTPClient = client
	fmt.Println("")
	fmt.Printf("This program will register the CONNCTD connector to all dS-Accounts, given in the file %s.\n", FILE_NAME)
	fmt.Println("")
	fmt.Printf("Reading file %s  ... ", FILE_NAME)
	readFile, err := os.Open(FILE_NAME)

	if err != nil {
		fmt.Println("ERROR")
		fmt.Println(err)
		fmt.Println("Program stopped")
		return
	}
	fmt.Println("OK")
	fileScanner := bufio.NewScanner(readFile)

	fileScanner.Split(bufio.ScanLines)

	fileScanner.Scan()
	firstLine := fileScanner.Text()
	if firstLine != COL_NAMES {
		fmt.Printf("wrong column names, must be %s\n", COL_NAMES)
		fmt.Println("Program stopped")
		return
	}

	accounts := []*accountRow{}

	for fileScanner.Scan() {
		newLine := fileScanner.Text()
		newAccount, err := getAccountRow(newLine)
		if err != nil {
			fmt.Printf("error, unable to read line: '%s' (%s)\n", newLine, err)
			fmt.Println("Program stopped")
			return
		}
		accounts = append(accounts, newAccount)

	}
	readFile.Close()
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
		return err
	}

	if err := os.WriteFile(filename, res, 0666); err != nil {
		return err
	}

	return nil

}

func saveReport(accounts []*accountRow, filename string) error {
	lines := []string{}

	for i := range accounts {
		if accounts[i].success {
			lines = append(lines, fmt.Sprintf("SUCCESS %s ", accounts[i].link))
		} else {
			lines = append(lines, fmt.Sprintf("FAIL    %s (%s)", accounts[i].link, accounts[i].err))
		}
	}
	f, err := os.Create(filename)
	if err != nil {

		return err
	}
	// remember to close the file
	defer f.Close()

	for _, line := range lines {
		_, err := fmt.Fprintln(f, line)
		if err != nil {
			return err
		}
	}

	return nil
}

func registerConnector(dSAccount *digitalstrom.Account, accountRow *accountRow) {
	fmt.Printf(" registering application '%s' at %s .... ", APP_NAME, accountRow.link)

	dSAccount.SetURL(accountRow.link)

	atoken, err := dSAccount.RegisterApplication(APP_NAME, accountRow.user, accountRow.secret)
	if err != nil {
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
	f, err := os.OpenFile("logs.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	digitalstrom.SetLogger(stdr.New(log.New(f, "", log.LstdFlags|log.Lshortfile)))
}
