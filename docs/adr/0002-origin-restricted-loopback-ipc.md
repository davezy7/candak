# Origin-Restricted Loopback IPC and Graceful Browser Fallback

Candak exposes a local loopback HTTP server on `127.0.0.1:41920` to receive intercepted downloads from its companion browser extension. To safeguard against malicious websites triggering downloads, requests are restricted to extension origins, and the extension falls back to native browser downloads if Candak is unreachable.

## Considered Options

* **Mandatory Custom API Token**: Require manual entry or exchange of an API secret between the app and extension settings. Adds setup friction for the user on first install.
* **Open Unrestricted Loopback**: Accept POST requests from any localhost client. Rejected due to security risks from drive-by web pages sending cross-origin loopback requests.
* **Origin-Restricted Loopback with Native Fallback**: Validate that requests bear `chrome-extension://` or `moz-extension://` origin schemes. If Candak is offline or fails health checks, the extension allows the browser's default download manager to complete the download.

## Consequences

* Seamless zero-configuration setup for the browser extension.
* No downloads are lost or blocked when the Candak desktop application is closed.
