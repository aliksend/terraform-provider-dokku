package dokkuclient

import "fmt"

func DoubleDashArg[T any](key string, value T) string {
	return fmt.Sprintf("--%s=%s", key, value)
}
