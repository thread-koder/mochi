package compute

import (
	"fmt"
	"strconv"

	"github.com/google/uuid"
)

func parseInt(s string) (int, error) {
	result, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("failed to parse integer: %w", err)
	}
	return result, nil
}

func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse UUID: %w", err)
	}
	return id, nil
}
