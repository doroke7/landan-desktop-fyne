#ifndef REOPEN_DARWIN_H
#define REOPEN_DARWIN_H

// InstallReopenHandler makes clicking the Dock icon call goReopen. Fyne does not handle macOS's "reopen" event,
// so without this a hidden window can only be brought back from the tray. Safe to call from any thread.
void InstallReopenHandler(void);

// Implemented in Go (reopen_darwin.go).
void goReopen(void);

#endif
