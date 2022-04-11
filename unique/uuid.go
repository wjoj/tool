package unique

import (
	uuid "github.com/satori/go.uuid"
)

func UUID() string {
	return uuid.NewV4().String()
}
