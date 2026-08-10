-- Sliding window rate limiter
-- KEYS[1] = redis key for this client (e.g. "rl:1.2.3.4")
-- ARGV[1] = current timestamp in ms
-- ARGV[2] = window size in ms (e.g. 60000 for 1 minute)
-- ARGV[3] = max requests allowed in window
-- ARGV[4] = trust multiplier (1.0 = normal, <1.0 = tighter limit for flagged clients)

local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local trust_multiplier = tonumber(ARGV[4])

local effective_limit = math.floor(limit * trust_multiplier)
if effective_limit < 1 then effective_limit = 1 end

local window_start = now - window

-- drop timestamps outside the window
redis.call("ZREMRANGEBYSCORE", key, 0, window_start)

local count = redis.call("ZCARD", key)

if count >= effective_limit then
    return {0, count, effective_limit}
end

redis.call("ZADD", key, now, now .. "-" .. math.random())
redis.call("PEXPIRE", key, window)

return {1, count + 1, effective_limit}
