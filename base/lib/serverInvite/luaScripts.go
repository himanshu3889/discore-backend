package serverInviteLib

import "github.com/redis/go-redis/v9"

// Consume the server invite using the cache
var consumeInviteScript = redis.NewScript(`
	local usageKey = KEYS[1]
	local maxUses = tonumber(ARGV[1])
	
	-- Get current uses, default to -1 if key doesn't exist; this thing need to be pre exist
	local currentUses = tonumber(redis.call("GET", usageKey) or "-1")
	if currentUses < 0 then
		return -1 -- Not found in cache
	end
	
	-- If maxUses is greater than 0, enforce the limit
	if maxUses > 0 and currentUses >= maxUses then
		return -429 -- Limit exceeded
	end
	
	-- Increment atomically
	local newCount = redis.call("INCR", usageKey)
	
	return newCount
`)

// Safely reverse an invite consumption (rollback)
var rollbackInviteScript = redis.NewScript(`
    local usageKey = KEYS[1]
    
    -- Get current uses
    local currentUses = redis.call("GET", usageKey)
    
    -- If key doesn't exist (expired/deleted), do nothing
    if not currentUses then
        return 0
    end
    
    -- If it's strictly greater than 0, decrement it
    if tonumber(currentUses) > 0 then
        return redis.call("DECR", usageKey)
    end
    
    -- If it's already 0, don't drop into negatives
    return tonumber(currentUses)
`)
