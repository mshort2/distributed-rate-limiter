-- Load test script for rate limiter
request = function()
    local client_id = "client_" .. math.random(1, 100)
    local body = string.format('{"client_id": "%s"}', client_id)
    
    return wrk.format("POST", "/check", {
        ["Content-Type"] = "application/json",
        ["X-Client-ID"] = client_id
    }, body)
end

response = function(status, headers, body)
    if status ~= 200 and status ~= 429 then
        print("Unexpected status: " .. status)
        print("Body: " .. body)
    end
end
