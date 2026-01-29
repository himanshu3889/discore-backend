package utils

import (
	"errors"
	"log"
	"strconv"
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
	SnowflakeNode *snowflake.Node
	once          sync.Once
)

// InitSnowflake initializes the generator with a Machine ID (e.g., 1)
// In a real distributed system, this ID comes from an env var unique to the pod/server
func InitSnowflake(machineID int64) {
	once.Do(func() {
		var err error
		SnowflakeNode, err = snowflake.NewNode(machineID)
		if err != nil {
			log.Fatalf("Failed to initialize snowflake node: %v", err)
		}
	})
}

// GenerateID returns a new ID
func GenerateSnowflakeID() snowflake.ID {
	return SnowflakeNode.Generate()
}

// isValidSnowflake checks if a string is a valid snowflake ID
func ValidSnowflakeID(id string) (snowflake.ID, error) {
	// Try to parse as uint64
	parsedID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, errors.New("Invalid ID")
	}

	// Create a snowflake ID from the parsed uint64
	snowflakeID := snowflake.ID(parsedID)

	// Validate by checking if it can extract basic components
	// Snowflake IDs should be > 0 and have reasonable timestamp
	if snowflakeID.Int64() > 0 && snowflakeID.Time() > 0 {
		return snowflakeID, nil
	}
	return 0, errors.New("Invalid ID")
}
