JSON = (loadfile "/var/lib/lua/json.lua")()
BASE64 = (loadfile "/var/lib/lua/base64.lua")()
function envoy_on_request(request_handle)
    local start_time = request_handle:timestamp()
    request_handle:headers():remove("x-jwt-claim-sub")
    local jwt_token = request_handle:headers():get("Authorization")
    if jwt_token == nil then
      request_handle:logDebug("Authorization header not found")
      return
    end
    local _, payload, _ = string.match(jwt_token, "^Bearer (.+)%.(.+)%.(.+)$")
    if payload == nil then
      request_handle:logDebug("Not a JWT token")
      return
    end
    request_handle:logDebug("payload: " .. payload)
    local payload_decoded = BASE64.decode(payload)
    local claims = JSON.decode(payload_decoded)
    request_handle:headers():add("x-jwt-claim-sub", claims["sub"])
    local end_time = request_handle:timestamp()
    request_handle:logDebug("Time " .. end_time - start_time)
end