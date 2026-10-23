# Browser Interception via Companion Extension and Loopback HTTP API

To intercept downloads initiated in external web browsers without invasive system proxying or requiring browser-specific native messaging hosts, Candak integrates with a companion browser extension (Manifest V3) that forwards download metadata to a local loopback HTTP server embedded inside Candak.

## Considered Options

* **System-wide HTTP/HTTPS Proxy**: Intercepts all operating system network traffic. Rejected because it requires installing custom self-signed root CA certificates, breaks strict HTTPS/HSTS, and incurs substantial performance overhead.
* **Native Messaging Host**: Requires registering OS-level JSON manifests into system application support directories for each browser. More complex deployment and fragile across browser variations.
* **Companion Extension with Local Loopback HTTP API**: The extension intercepts `chrome.downloads.onDeterminingFilename`, cancels the browser's native download, and dispatches an HTTP POST payload containing URL, Cookies, and Referer to `127.0.0.1:<port>`. Simple, robust, and sandbox-compliant.

## Consequences

* Candak must run a lightweight HTTP daemon on localhost while the application is active.
* Requests to the loopback server should include security safeguards (origin verification or a local token) to prevent arbitrary malicious web pages from initiating downloads through Candak.
