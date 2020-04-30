# Capturoo command line tool

## Runtime Configuration
Optionally, use `CAPTUROO_CLI_ENDPOINT` environment variable to override the predefined endpoint.

### Example for testing
```bash
ENDPOINT=http://localhost:8080 capturoo login --email
```

## Build
Replace `<endpoint>` with the API endpoint.

```bash
make
```
