package utils

import (
	"werewolves-go/data"
	"werewolves-go/types"

	"github.com/anthdm/hollywood/actor"
)

func GetWerewolves(users map[string]data.Client, clients map[string]*actor.PID) []*actor.PID {

	var pidList []*actor.PID
	for cAddr, data := range users {
		if data.Role == "werewolf" {
			pidList = append(pidList, clients[cAddr])
		}
	}

	return pidList
}

func GetListofUsernames(users map[string]data.Client) []string {
	var userList []string
	for _, data := range users {
		userList = append(userList, data.Name)
	}

	return userList
}

func GetCAddrFromUsername(users map[string]data.Client, username string) string {
	for caddr, user := range users {
		if user.Name == username {
			return caddr
		}
	}

	return ""
}

func FormatMessageResponseFromServer(message string) *types.Message {
	msgResponse := &types.Message{
		Username: "server/primary",
		Msg:      message,
	}

	return msgResponse
}

func IsUsernameAllowed(username string, users map[string]data.Client) bool {
	for _, user := range users {
		if user.Name == username {
			return true
		}
	}

	return false
}
