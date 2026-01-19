# ASCOM Alpaca API Interface Rules

**Full Specification**: `plugins/examples/ascom-alpaca-simulator/ASCOM_ALPACA_API_SPEC.json`
**Swagger UI**: http://localhost:32323/swagger
**Official Documentation**: https://ascom-standards.org/Developer/Alpaca.htm

## CRITICAL RULES

### 1. HTTP Methods

- **GET**: For reading properties/state (e.g., position, status)
- **PUT**: For ALL commands and property changes (e.g., slew, park, set tracking)

### 2. Request Format for PUT Requests

**IMPORTANT**: PUT requests MUST use `multipart/form-data` or `application/x-www-form-urlencoded`

- **ALL parameters** (including ClientID and ClientTransactionID) MUST be in the request body
- **NO query parameters** should be used with PUT requests
- **Content-Type header** must be set to `multipart/form-data` or `application/x-www-form-urlencoded`

❌ **WRONG**:
```bash
curl -X PUT "http://localhost/api/v1/telescope/0/park?ClientID=1&ClientTransactionID=1"
```

✅ **CORRECT**:
```bash
curl -X PUT "http://localhost/api/v1/telescope/0/park" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "ClientID=1&ClientTransactionID=1"
```

### 3. Required Parameters

**Every request** (GET or PUT) MUST include:
- `ClientID`: Integer (1-4294967295) - Unique client identifier
- `ClientTransactionID`: Integer (1-4294967295) - Sequential transaction counter

**For GET requests**: Use query parameters
**For PUT requests**: Use form data in request body

### 4. Response Format

All responses are JSON with this structure:
```json
{
  "ClientTransactionID": 1,
  "ServerTransactionID": 123,
  "ErrorNumber": 0,
  "ErrorMessage": "",
  "Value": <result_if_applicable>
}
```

- `ErrorNumber`: 0 = success, non-zero = error
- `ErrorMessage`: Empty on success, error description on failure

## Common Telescope Endpoints

### Connect/Disconnect

```bash
# Connect
PUT /api/v1/telescope/0/connected
Content-Type: application/x-www-form-urlencoded
Body: Connected=true&ClientID=1&ClientTransactionID=1

# Disconnect
PUT /api/v1/telescope/0/connected
Content-Type: application/x-www-form-urlencoded
Body: Connected=false&ClientID=1&ClientTransactionID=1

# Check connection status (GET)
GET /api/v1/telescope/0/connected?ClientID=1&ClientTransactionID=1
```

### Park/Unpark

```bash
# Park
PUT /api/v1/telescope/0/park
Content-Type: application/x-www-form-urlencoded
Body: ClientID=1&ClientTransactionID=1

# Unpark
PUT /api/v1/telescope/0/unpark
Content-Type: application/x-www-form-urlencoded
Body: ClientID=1&ClientTransactionID=1

# Check if parked (GET)
GET /api/v1/telescope/0/atpark?ClientID=1&ClientTransactionID=1
```

### Slew Commands

```bash
# Slew to coordinates (async)
PUT /api/v1/telescope/0/slewtocoordinatesasync
Content-Type: application/x-www-form-urlencoded
Body: RightAscension=12.5&Declination=45.0&ClientID=1&ClientTransactionID=1

# Abort slew
PUT /api/v1/telescope/0/abortslew
Content-Type: application/x-www-form-urlencoded
Body: ClientID=1&ClientTransactionID=1

# Check if slewing (GET)
GET /api/v1/telescope/0/slewing?ClientID=1&ClientTransactionID=1
```

### Tracking

```bash
# Set tracking on/off
PUT /api/v1/telescope/0/tracking
Content-Type: application/x-www-form-urlencoded
Body: Tracking=true&ClientID=1&ClientTransactionID=1

# Get tracking status (GET)
GET /api/v1/telescope/0/tracking?ClientID=1&ClientTransactionID=1
```

### Position Queries (GET)

```bash
# Right Ascension (hours)
GET /api/v1/telescope/0/rightascension?ClientID=1&ClientTransactionID=1

# Declination (degrees)
GET /api/v1/telescope/0/declination?ClientID=1&ClientTransactionID=1

# Altitude (degrees)
GET /api/v1/telescope/0/altitude?ClientID=1&ClientTransactionID=1

# Azimuth (degrees)
GET /api/v1/telescope/0/azimuth?ClientID=1&ClientTransactionID=1
```

## Go Implementation Patterns

### For PUT Requests with Form Data

```go
func callASCOMPut(endpoint string, params map[string]string) error {
    apiURL := fmt.Sprintf("%s%s", ascomBaseURL, endpoint)
    
    // Build form data
    formData := url.Values{}
    for key, value := range params {
        formData.Set(key, value)
    }
    formData.Set("ClientID", "1")
    formData.Set("ClientTransactionID", fmt.Sprintf("%d", time.Now().Unix()))
    
    // Create request with form body
    req, err := http.NewRequest(http.MethodPut, apiURL, strings.NewReader(formData.Encode()))
    if err != nil {
        return err
    }
    
    // CRITICAL: Set Content-Type header
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    body, _ := io.ReadAll(resp.Body)
    log.Printf("Response: %s", string(body))
    return nil
}
```

### For GET Requests with Query Parameters

```go
func callASCOMGet(endpoint string) (interface{}, error) {
    apiURL := fmt.Sprintf("%s%s", ascomBaseURL, endpoint)
    
    // Build query parameters
    params := url.Values{}
    params.Set("ClientID", "1")
    params.Set("ClientTransactionID", fmt.Sprintf("%d", time.Now().Unix()))
    
    // Append query string
    fullURL := apiURL + "?" + params.Encode()
    
    resp, err := http.Get(fullURL)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    return result["Value"], nil
}
```

## Common Error Codes

| ErrorNumber | Meaning |
|-------------|---------|
| 0 | Success |
| 1024 | Not implemented |
| 1025 | Invalid value |
| 1026 | Value not set |
| 1027 | Not connected |
| 1031 | Invalid operation |
| 1032 | Action not allowed (e.g., slew while parked) |
| 1033 | Not in cache |

## Device Numbering

- Devices are zero-indexed
- Device number 0 is typically the first/default device
- Path format: `/api/v1/{device_type}/{device_number}/{property_or_method}`

Examples:
- `/api/v1/telescope/0/park`
- `/api/v1/camera/0/exposure`
- `/api/v1/focuser/0/position`

## Testing Commands

```bash
# Test connection
curl -X PUT "http://localhost:32323/api/v1/telescope/0/connected" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "Connected=true&ClientID=1&ClientTransactionID=1"

# Test park
curl -X PUT "http://localhost:32323/api/v1/telescope/0/park" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "ClientID=1&ClientTransactionID=1"

# Test get position
curl "http://localhost:32323/api/v1/telescope/0/rightascension?ClientID=1&ClientTransactionID=1"
```

## References

- Full OpenAPI 3.0 spec: `plugins/examples/ascom-alpaca-simulator/ASCOM_ALPACA_API_SPEC.json`
- ASCOM Alpaca Documentation: https://ascom-standards.org/Developer/Alpaca.htm
- Simulator Swagger UI: http://localhost:32323/swagger

## When to Consult This Document

- When implementing any ASCOM API calls in Go code
- When debugging communication with the ASCOM simulator
- When adding new device control features
- When the API returns validation errors
- Before creating any new command-bridge handlers
