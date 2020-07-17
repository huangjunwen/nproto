package stanmsg

import (
	"fmt"
)

func subjectFormat(subjectPrefix, subject string) string {
	return fmt.Sprintf("%s.%s", subjectPrefix, subject)
}
