package natsrpc

import (
	"fmt"
	"strings"
)

func subjectFormat(subjectPrefix, svcName, methodName string) (subject string) {
	return fmt.Sprintf("%s.%s.%s", subjectPrefix, svcName, methodName)
}

func subjectParser(subjectPrefix, svcName string) func(subject string) (ok bool, methodName string) {
	prefix := fmt.Sprintf("%s.%s.", subjectPrefix, svcName)
	return func(subject string) (bool, string) {
		methodName := strings.TrimPrefix(subject, prefix)
		if methodName == subject {
			return false, ""
		}
		return true, methodName
	}
}
