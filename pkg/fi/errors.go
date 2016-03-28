package fi

import (
	"fmt"
)

func MissingValueError(message string) error {
	return fmt.Errorf("%s", message)
}

func InvalidChangeError(message string, actual, expected interface{}) error {
	return fmt.Errorf("Invalid change: %s current=%q, desired=%q", message, DebugPrint(actual), DebugPrint(expected))
}

func RequiredField(key string) error {
	return fmt.Errorf("Field is required: %s", key)
}

func CannotChangeField(key string) error {
	return fmt.Errorf("Field cannot be changed: %s", key)
}
