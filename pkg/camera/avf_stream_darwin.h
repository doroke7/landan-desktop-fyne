//go:build darwin && cgo

#ifndef AVF_STREAM_DARWIN_H
#define AVF_STREAM_DARWIN_H

#include <stdint.h>

// Frame kinds handed to goCameraFrame.
#define AVF_FRAME_PREVIEW 0  // scaled to the preview size, RGBA
#define AVF_FRAME_SNAPSHOT 1 // full size, RGBA

// Implemented in Go (camera_darwin.go). The pixel data is only valid during the call.
extern void goCameraFrame(uintptr_t iHandle, int iKind, unsigned char *aData, int iWidth, int iHeight, int iStride);
// Implemented in Go. cMessage is only valid during the call.
extern void goCameraError(uintptr_t iHandle, char *cMessage);

// avf_stream_start opens the camera and starts, all at once:
//   - a live preview   (goCameraFrame, AVF_FRAME_PREVIEW, iPreviewFps frames per second),
//   - an mp4 recording (cRecordPath, H.264 at iBitrate bits per second),
//   - snapshots        (goCameraFrame, AVF_FRAME_SNAPSHOT, every dSnapshotSeconds; 0 disables them).
//
// cDevice is "default", a zero-based index, or part of the camera's name.
// It returns an opaque stream on success. On failure it returns NULL and *oError is a malloc'd message the caller must free.
void *avf_stream_start(const char *cDevice, int iWidth, int iHeight, int iFps, int iBitrate,
                       int iPreviewWidth, int iPreviewHeight, int iPreviewFps,
                       const char *cRecordPath, double dSnapshotSeconds,
                       uintptr_t iHandle, int iTimeoutMs, char **oError);

// avf_stream_stop stops the camera, waits until the mp4 is finalized, and frees the stream.
// No callback runs after it returns.
void avf_stream_stop(void *pStream);

#endif
