//go:build windows && cgo

#ifndef MF_STREAM_WINDOWS_H
#define MF_STREAM_WINDOWS_H

#include <stdint.h>

// Frame kinds handed to goCameraFrame.
#define MF_FRAME_PREVIEW 0  // scaled to the preview size, RGBA
#define MF_FRAME_SNAPSHOT 1 // full size, RGBA

// Implemented in Go (stream_windows.go). The pixel data is only valid during the call.
extern void goCameraFrame(uintptr_t iHandle, int iKind, unsigned char *aData, int iWidth, int iHeight, int iStride);
// Implemented in Go. cMessage is only valid during the call.
extern void goCameraError(uintptr_t iHandle, char *cMessage);

// mf_stream_start opens the camera via Media Foundation and starts, all at once:
//   - a live preview   (goCameraFrame, MF_FRAME_PREVIEW, ~iPreviewFps frames per second),
//   - an mp4 recording (cRecordPath, H.264 at iBitrate bits per second) - empty path disables it,
//   - snapshots        (goCameraFrame, MF_FRAME_SNAPSHOT, every dSnapshotSeconds; 0 disables them).
//
// cDevice is "" or "default" for the first camera, a zero-based index, or part of the camera's
// friendly name (case-insensitive).
//
// Capture and encoding run on a dedicated worker thread owned by the returned stream; this call
// itself only opens and configures the device and blocks until that succeeds or fails.
// On failure it returns NULL and *oError is a malloc'd message the caller must free.
void *mf_stream_start(const char *cDevice, int iWidth, int iHeight, int iFps, int iBitrate,
                       int iPreviewWidth, int iPreviewHeight, int iPreviewFps,
                       const char *cRecordPath, double dSnapshotSeconds,
                       uintptr_t iHandle, char **oError);

// mf_stream_stop stops the camera, waits until the worker thread has exited and the mp4 (if any)
// is finalized, and frees the stream. No callback runs after it returns.
void mf_stream_stop(void *pStream);

#endif
