package utils

import (
	"werewolves-go/data"
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
