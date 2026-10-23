# System Tray Daemon, Standalone Mini Prompt, and Desktop Notifications

To match the non-intrusive workflow of dedicated download managers (like IDM), Candak persists as a system tray daemon with `HideWindowOnClose: true`. Intercepted downloads display a compact, standalone Mini Prompt window, emit native OS desktop notifications on start and completion, and return directly to the tray.

## Considered Options

* **Keep Full Dashboard Window Open**: Revealing the 1080x720 window with the entire download queue on every browser link click. Rejected because it interrupts the user's web browsing and clutters the screen.
* **Secondary OS Process**: Spawning a separate subprocess binary for each download dialog. Rejected because Wails v2 webview embedding causes duplicate dock icons and complicated inter-process state synchronization.
* **System Tray Daemon with Dynamic Mini Prompt**: The main application hides to the system tray on close. When a download is captured from the browser, Candak resizes to compact dimensions (520x380) in standalone prompt mode. Upon starting the download, a native desktop notification confirms the event, and the window returns to the tray.

## Consequences

* The application stays active in the system tray when closed.
* Native OS notifications provide non-intrusive feedback for download start and completion.
* Full application dashboard is accessible on demand via the system tray icon.
