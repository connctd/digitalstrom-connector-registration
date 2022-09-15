# digitalSTROM connector registration tool

The aim of this tool is to register an application 'foresight-connectd' to multiple digitalStrom systems and extracting the application tokens which are needed for the application login. With the help of the generated token, a connector of the connctd platform is able to establish a connection without the use of user credentials. 

# Usage
## Prepare a file with account data
The tool searches for a file named ```accounts.csv```. This file contains a list of account data ```url```, ```username```, ```password```, separated by the symbol ```;```.
Make sure to use the exact file name as well as the exact column order and column names (see file ```example.csv```)

## Build

    make

### Windows

    make windows

### MacOs

    make macos

### Linux

    make linux

### Other Architectures and Operating Systems

In order to build the code for other operating systems and archtectures, type the command ```go tool dist list``` for a list of pairs ```<GOOS>/<GOARCH>```. If you want to build the tool for a linux system on an arm architecture (linux/arm), type the build command ```env GOOS=linux GOARCH=arm go build```.  

## Run

Run the tool. It searches for the prepared file ```accounts.csv``` and registers an application 'foresight-connectd' to all digitalStrom systems listed. When the tool has finished, it generates to files ```report.log``` and ```tokens.json```. The file ```report.log``` gives an overview of the success or fail status for each system as well as the error description when an error occured during the process. The file ```tokens.json``` contains the tokens for each successful registration (failed registration will be ignored in this file)

