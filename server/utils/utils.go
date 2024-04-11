package utils

import (
	"werewolves-go/data"

	"github.com/anthdm/hollywood/actor"
)

func DoesWerewolfExist(users map[string]data.Client) bool {
	for _, value := range users {
		if value.Role == "werewolf" {
			return true
		}
	}

	return false
}

func DoesWitchExist(users map[string]data.Client) bool {
	for _, value := range users {
		if value.Role == "witch" {
			return true
		}
	}

	return false
}

func GetWerewolves(users map[string]data.Client, clients map[string]*actor.PID) []*actor.PID {

	var pidList []*actor.PID
	for cAddr, data := range users {
		if data.Role == "werewolf" {
			pidList = append(pidList, clients[cAddr])
		}
	}

	return pidList
}
