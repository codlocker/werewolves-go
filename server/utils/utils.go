package utils

import (
	"fmt"
	"werewolves-go/data"
	"werewolves-go/types"

	"github.com/anthdm/hollywood/actor"
)

func AreWerewolvesAlive(users map[string]*data.Client) bool {
	for _, user := range users {
		if user.Status && user.Role == "werewolf" {
			return true
		}
	}

	return false
}

func AreTownspersonAlive(users map[string]*data.Client) bool {
	for _, user := range users {
		if user.Status && user.Role == "townsperson" {
			return true
		}
	}

	return false
}

func GetAliveWerewolves(users map[string]*data.Client, clients map[string]*actor.PID) []*actor.PID {

	var pidList []*actor.PID
	for cAddr, data := range users {
		if data.Role == "werewolf" && data.Status {
			pidList = append(pidList, clients[cAddr])
		}
	}

	return pidList
}

func GetAliveTownperson(users map[string]*data.Client, clients map[string]*actor.PID) []*actor.PID {

	var pidList []*actor.PID
	for cAddr, data := range users {
		if data.Status {
			pidList = append(pidList, clients[cAddr])
		}
	}

	return pidList
}

func GetListofUsernames(users map[string]*data.Client) []string {
	var userList []string
	for _, data := range users {
		if data.Status {
			userList = append(userList, data.Name)
		}
	}

	return userList
}

func GetCAddrFromUsername(users map[string]*data.Client, username string) string {
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

func IsUsernameAllowed(username string, users map[string]*data.Client) bool {
	for _, user := range users {
		if user.Name == username {
			return true
		}
	}

	return false
}

func SetUpRoles(users map[string]*data.Client, werewolves map[string]*data.Client, number_of_werewolves int) {

	user_names := GetListofUsernames(users)

	// Set up werewolf
	for i := 0; i < number_of_werewolves; i++ {
		caddr := GetCAddrFromUsername(users, user_names[i])
		if users[caddr].Role == "" {
			if entry, ok := users[caddr]; ok {
				entry.Role = "werewolf"
				users[caddr] = entry
				werewolves[caddr] = entry
			}
		}
	}

	// Set up townsperson
	for caddr, user := range users {
		if user.Role == "" {
			user.Role = "townsperson"
			users[caddr] = user
		}
	}
}

func PrintUsers(users map[string]*data.Client) {
	fmt.Println("Print Users")
	for user, data := range users {
		fmt.Printf(
			"%v has been assigned %v username, has role %v with alive status %v\n",
			user,
			data.Name,
			data.Role,
			data.Status)
	}
}
