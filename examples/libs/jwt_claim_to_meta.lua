JSON = (loadfile "/var/lib/lua/json.lua")()
BASE64 = (loadfile "/var/lib/lua/base64.lua")()
function envoy_on_request(request_handle)
    local start_time = os.time()
    request_handle:headers():remove("x-jwt-claim-sub")
    local jwt_token = request_handle:headers():get("Authorization")
    if jwt_token == nil then
      request_handle:logInfo("Authorization header not found")
      return
    end
    local header, payload, signature = string.match(jwt_token, "^Bearer (.+)%.(.+)%.(.+)$")
    if payload == nil then
      request_handle:logInfo("Not a JWT token")
      return
    end
    request_handle:logInfo("payload: " .. payload)
    local payload_decoded = BASE64.decode(payload)
    local claims = JSON.decode(payload_decoded)
    request_handle:logInfo("x-jwt-claim-sub: " .. claims["sub"])
    request_handle:headers():add("x-jwt-claim-sub", claims["sub"])
    local end_time = os.time()
    request_handle:logInfo("Time" .. os.difftime(end_time, start_time))
end