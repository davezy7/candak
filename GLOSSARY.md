# Candak Download Manager

A desktop download manager featuring concurrent chunking, resume support, and browser download interception.

## Language

**Browser Interceptor**:
A companion browser extension mechanism that detects and intercepts browser download events before the native browser download manager starts.
_Avoid_: Browser hook, download hijack

**Download Prompt**:
The pre-download modal interface presenting the intercepted or pasted URL, probed file size, detected filename, destination directory, and Download/Cancel action buttons.
_Avoid_: Add modal, popup window

**Mini Prompt Window**:
A compact standalone popup window dedicated exclusively to download confirmation while the main manager remains hidden in the system tray.
_Avoid_: Mini app, sub-window

**System Tray Daemon**:
The background execution state of Candak resident in the OS menu bar or system tray with the main library dashboard closed.
_Avoid_: Background process, tray hook

**Hide-to-Tray**:
The window lifecycle behavior where closing the main window hides it to the system tray instead of terminating the application.
_Avoid_: Soft close, minimize to bar

**Auto-Probe**:
Immediate HTTP inspection (HEAD/GET range capabilities, content length, MIME type, and filename headers) triggered automatically whenever a URL is ingested or pasted.
_Avoid_: URL scan, link check

**Clipboard Watcher**:
An opt-in background monitor observing the OS clipboard for file URLs matching specified downloadable file extensions.
_Avoid_: Clipboard sniffer

**Self-Origin Suppression**:
Filtering mechanism that ignores clipboard changes originating from Candak's own UI actions to prevent recursive prompt triggers.
_Avoid_: Loop breaker, duplicate ignore
