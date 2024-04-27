## WEREWOLVES GAME 

This repo creates the well known werewolves game using Go Programming language. The program is architectured around using Actor model for defining the server and client programs. The communication mode is using gRPC which supports bidirectional streaming for RPCs.

## STACK

1. [Go](https://go.dev/)
2. [Hollywood](https://github.com/anthdm/hollywood)
3. [dRPC](https://github.com/storj/drpc)

## Installation requirements (Linux)

1. Downloading Golang
  - Wget go files ```wget https://go.dev/dl/go1.22.2.linux-amd64.tar.gz```
  - Run ```sudo rm -rf /usr/local/go```to remove any existing installations
  - Run ```export PATH=$PATH:/usr/local/go/bin```
  - Run ```go version``` (If the go version gets listed, your installation worked fine)

2. Get Werewolves repo
  - Run ``` git clone https://github.com/codlocker/werewolves-go.git```
  - Or you can use Download ZIP or Download tar file in case git isn't installed in the system

## How to run?
- Ensure you are in the werewolf code repo (Run ```cd werewolves-go/```)
- Run ```make build```
- The client and server builds are created in the [/bin/](./bin/) folder.
- Clean existing builds by running ```make clean```.

- #### Run Server first (Everything needs to run in werewolves-go path)
  - To run server execute command
    - Execute: ```cd bin/```
    - Execute: ```./server```
- #### Run Clients next (Everything needs to run in werewolves-go path)
  - To run client execude command (Open a new terminal window for each client you want to run)
    - Execute: ```cd bin/```
    - Execute (**change username** to a new name in every client window): ```./client -username=<username>```
      - Examples: ```./client username=a```
      - Examples: ```./client username=b```
      - Examples: ```./client username=c```
      - Examples: ```./client username=d```