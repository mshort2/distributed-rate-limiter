-- Advanced load test with multiple scenarios
local counter = 0
local clients = {}

-- Generate 1000 unique client IDs
for i = 1, 1000 do
    table.insert(clients, "client_" .. i)
end

request = function()
    counter = counter + 1
    local client_id = clients[(counter % #clients) + 1]
    
    -- Mix of different request patterns
    local headers = {}
    if counter % 3 == 0 then
        headers["X-API-Key"] = client_id
    else
        headers["X-Client-ID"] = client_id
    end
    
    headers["Content-Type"] = "application/json"
    
    return wrk.format("POST", "/check", headers, "")
end

response = function(status, headers, body)
    -- Track response codes
    if status == 200 then
        -- Allowed
    elseif status == 429 then
        -- Rate limited
    else
        print("Unexpected status: " .. status)
        print("Body: " .. body)
    end
end

done = function(summary, latency, requests)
    print("\nLoad Test Results:")
    print("Total requests: " .. summary.requests)
    --print("Successful requests: " .. (summary.requests - summary.errors))
    local total_errors = 0
    for k, v in pairs(summary.errors) do
        total_errors = total_errors + v
    end
    print("Errors: " .. total_errors)
    print("Error breakdown:")
    for k, v in pairs(summary.errors) do
        print("  " .. tostring(k) .. ": " .. tostring(v))
    end
    print("Average latency: " .. latency.mean .. "ms")
    print("99th percentile: " .. latency:percentile(99) .. "ms")
    print("Max latency: " .. latency.max .. "ms")
end