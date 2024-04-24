## WEREWOLVES GAME 

This repo creates the well known werewolves game using Go Programming language. The program is architectured around using Actor model for defining the server and client programs. The communication mode is using gRPC which supports bidirectional streaming for RPCs.

## STACK

1. [Go](https://go.dev/)
2. [Hollywood](https://github.com/anthdm/hollywood)
3. [dRPC](https://github.com/storj/drpc)

## How to run?

- Run ```make build```
- The client and server builds are created in the [/bin/](./bin/) folder.
- Clean exisitng builds by running ```make clean```.