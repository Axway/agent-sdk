package notify

import "fmt"

type mimeMap map[string]string

func (m *mimeMap) String() string {
	mimeString := ""

	for key, value := range *m {
		mimeString += fmt.Sprintf("%s: %s;\n", key, value)
	}
	return mimeString
}
