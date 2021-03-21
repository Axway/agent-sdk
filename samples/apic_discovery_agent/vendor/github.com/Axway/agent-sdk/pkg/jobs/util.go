package jobs

import "github.com/google/uuid"

func newUUID() string {
	id, _ := uuid.NewRandom()
	return id.String()
}
