//go:build windows && cgo

// Windows Media Foundation backend for pkg/camera.
//
// NOTE (written without a Windows machine to build/run against - verify these on real hardware):
//   - Frame rate: the source reader is left at the native frame rate the camera reports for the
//     chosen resolution (see choose_media_type); iFps is only used for the recording's declared
//     frame rate/timestamps and may not match the camera's actual capture rate.
//   - Row stride: read from MF_MT_DEFAULT_STRIDE when the driver sets it, else assumed to be a
//     tightly packed iWidth*4 top-down buffer. A negative value (bottom-up DIB) is only sign-fixed,
//     not flipped - if a real camera reports bottom-up frames, the image will come out upside down
//     and bgra_to_rgba/bgra_downscale_to_rgba need a per-row reversal added.
//
// One capture session feeds three things from a single Media Foundation source reader stream:
// a small RGBA preview (goCameraFrame/MF_FRAME_PREVIEW), an H.264 mp4 via IMFSinkWriter, and
// periodic full-size RGBA snapshots (goCameraFrame/MF_FRAME_SNAPSHOT, JPEG-encoded on the Go side).
// Everything - device enumeration through teardown - runs on one dedicated worker thread so the
// whole session stays in a single COM apartment (MTA); see capture_thread_proc.

#define COBJMACROS

#include <windows.h>
#include <mfapi.h>
#include <mfidl.h>
#include <mfreadwrite.h>
#include <mferror.h>
#include <process.h>
#include <stdio.h>
#include <stdlib.h>
#include <wchar.h>

#include "mf_stream_windows.h"

typedef struct MFStream {
	HANDLE hThread;
	HANDLE hReadyEvent; // signaled once setup finished, success or failure
	volatile LONG stopFlag;
	uintptr_t handle;

	// config, written once before the worker thread starts, read-only afterwards
	char *sDevice;
	int width, height, fps, bitrate;
	int previewWidth, previewHeight, previewFps;
	wchar_t *wRecordPath; // NULL disables recording
	double snapshotSeconds;

	// setup result, written by the worker thread before signaling hReadyEvent
	HRESULT setupResult;
	char *setupError; // malloc'd on failure; ownership moves to mf_stream_start's caller

	// only ever touched by the worker thread
	IMFSourceReader *pReader;
	IMFMediaSource *pSource;
	IMFSinkWriter *pWriter;
	DWORD writerStreamIndex;
	UINT32 frameWidth, frameHeight, frameStride;
	double previewIntervalMs, snapshotIntervalMs;
	LONGLONG frameDuration100ns;
} MFStream;

static char *format_hresult(const char *sPrefix, HRESULT hr) {
	char *cMessage = (char *)malloc(256);
	if (cMessage != NULL) {
		snprintf(cMessage, 256, "%s (hr=0x%08lX)", sPrefix, (unsigned long)hr);
	}
	return cMessage;
}

static wchar_t *utf8_to_wide(const char *cText) {
	if (cText == NULL) return NULL;
	int nLen = MultiByteToWideChar(CP_UTF8, 0, cText, -1, NULL, 0);
	if (nLen <= 0) return NULL;
	wchar_t *wText = (wchar_t *)malloc(sizeof(wchar_t) * (size_t)nLen);
	if (wText == NULL) return NULL;
	MultiByteToWideChar(CP_UTF8, 0, cText, -1, wText, nLen);
	return wText;
}

static BOOL wide_contains_ci(const wchar_t *wHaystack, const wchar_t *wNeedle) {
	if (wHaystack == NULL || wNeedle == NULL || *wNeedle == L'\0') return FALSE;
	size_t nHay = wcslen(wHaystack), nNeedle = wcslen(wNeedle);
	if (nNeedle > nHay) return FALSE;
	for (size_t i = 0; i + nNeedle <= nHay; i++) {
		if (_wcsnicmp(wHaystack + i, wNeedle, nNeedle) == 0) return TRUE;
	}
	return FALSE;
}

// pick_device selects a video capture device: "" / "default" picks the first one, a plain integer
// picks by zero-based index, anything else matches (case-insensitively) against a substring of the
// device's friendly name - the same convention documented on Options.Device for the AVFoundation
// backend.
static HRESULT pick_device(const char *cDevice, IMFActivate **ppActivate) {

	IMFAttributes *pAttr = NULL;
	IMFActivate **ppDevices = NULL;
	UINT32 nCount = 0;
	IMFActivate *pChosen = NULL;
	HRESULT hr;

	hr = MFCreateAttributes(&pAttr, 1);
	if (SUCCEEDED(hr)) {
		hr = IMFAttributes_SetGUID(pAttr, &MF_DEVSOURCE_ATTRIBUTE_SOURCE_TYPE,
		                            &MF_DEVSOURCE_ATTRIBUTE_SOURCE_TYPE_VIDCAP_GUID);
	}
	if (SUCCEEDED(hr)) {
		hr = MFEnumDeviceSources(pAttr, &ppDevices, &nCount);
	}
	if (SUCCEEDED(hr) && nCount == 0) {
		hr = E_FAIL;
	}

	if (SUCCEEDED(hr)) {
		BOOL bEmpty = (cDevice == NULL || cDevice[0] == '\0' || _stricmp(cDevice, "default") == 0);
		if (bEmpty) {
			pChosen = ppDevices[0];
			IMFActivate_AddRef(pChosen);
		} else {
			char *pEnd = NULL;
			long nIndex = strtol(cDevice, &pEnd, 10);
			if (pEnd != cDevice && *pEnd == '\0' && nIndex >= 0 && (UINT32)nIndex < nCount) {
				pChosen = ppDevices[nIndex];
				IMFActivate_AddRef(pChosen);
			} else {
				wchar_t *wNeedle = utf8_to_wide(cDevice);
				if (wNeedle != NULL) {
					for (UINT32 i = 0; i < nCount && pChosen == NULL; i++) {
						WCHAR *wName = NULL;
						UINT32 nNameLen = 0;
						if (SUCCEEDED(IMFActivate_GetAllocatedString(ppDevices[i], &MF_DEVSOURCE_ATTRIBUTE_FRIENDLY_NAME,
						                                              &wName, &nNameLen))) {
							if (wide_contains_ci(wName, wNeedle)) {
								pChosen = ppDevices[i];
								IMFActivate_AddRef(pChosen);
							}
							CoTaskMemFree(wName);
						}
					}
					free(wNeedle);
				}
			}
		}
		if (pChosen == NULL) hr = E_FAIL;
	}

	if (ppDevices != NULL) {
		for (UINT32 i = 0; i < nCount; i++) {
			if (ppDevices[i] != NULL) IMFActivate_Release(ppDevices[i]);
		}
		CoTaskMemFree(ppDevices);
	}
	if (pAttr != NULL) IMFAttributes_Release(pAttr);

	if (SUCCEEDED(hr)) *ppActivate = pChosen;
	return hr;
}

// choose_media_type picks the native type on the reader's first video stream whose frame size is
// closest to (iWidth, iHeight), and returns a copy of it with the subtype switched to RGB32 - the
// video processor enabled via MF_SOURCE_READER_ENABLE_VIDEO_PROCESSING then converts the camera's
// native format (MJPEG/NV12/YUY2/...) to RGB32 for us at that same frame size.
static HRESULT choose_media_type(IMFSourceReader *pReader, int iWidth, int iHeight, IMFMediaType **ppOutType) {

	HRESULT hr = S_OK;
	DWORD i = 0;
	IMFMediaType *pBest = NULL;
	UINT64 nBestDiff = 0;
	BOOL bHaveBest = FALSE;

	for (;;) {
		IMFMediaType *pType = NULL;
		hr = IMFSourceReader_GetNativeMediaType(pReader, (DWORD)MF_SOURCE_READER_FIRST_VIDEO_STREAM, i, &pType);
		if (hr == MF_E_NO_MORE_TYPES) {
			hr = S_OK;
			break;
		}
		if (FAILED(hr)) break;

		UINT32 cx = 0, cy = 0;
		if (SUCCEEDED(MFGetAttributeSize((IMFAttributes *)pType, &MF_MT_FRAME_SIZE, &cx, &cy))) {
			INT64 dx = (INT64)cx - (INT64)iWidth;
			INT64 dy = (INT64)cy - (INT64)iHeight;
			UINT64 nDiff = (UINT64)(dx * dx + dy * dy);
			if (!bHaveBest || nDiff < nBestDiff) {
				if (pBest != NULL) IMFMediaType_Release(pBest);
				pBest = pType;
				IMFMediaType_AddRef(pBest);
				nBestDiff = nDiff;
				bHaveBest = TRUE;
			}
		}
		IMFMediaType_Release(pType);
		i++;
	}

	if (FAILED(hr)) {
		if (pBest != NULL) IMFMediaType_Release(pBest);
		return hr;
	}
	if (!bHaveBest) return E_FAIL;

	IMFMediaType *pNewType = NULL;
	hr = MFCreateMediaType(&pNewType);
	if (SUCCEEDED(hr)) hr = IMFMediaType_CopyAllItems(pBest, (IMFAttributes *)pNewType);
	IMFMediaType_Release(pBest);
	if (FAILED(hr)) {
		if (pNewType != NULL) IMFMediaType_Release(pNewType);
		return hr;
	}

	hr = IMFMediaType_SetGUID(pNewType, &MF_MT_SUBTYPE, &MFVideoFormat_RGB32);
	if (SUCCEEDED(hr)) hr = IMFMediaType_SetUINT32(pNewType, &MF_MT_INTERLACE_MODE, MFVideoInterlace_Progressive);
	if (FAILED(hr)) {
		IMFMediaType_Release(pNewType);
		return hr;
	}

	*ppOutType = pNewType;
	return S_OK;
}

// setup_sink_writer opens an mp4 sink writer that encodes the RGB32 frames we feed it (cx x cy) into
// H.264 at iBitrate bits/second, iFps frames/second.
static HRESULT setup_sink_writer(const wchar_t *wPath, UINT32 cx, UINT32 cy, int iFps, int iBitrate,
                                  IMFSinkWriter **ppWriter, DWORD *pStreamIndex) {

	IMFAttributes *pSinkAttr = NULL;
	IMFMediaType *pOutType = NULL;
	IMFMediaType *pInType = NULL;
	IMFSinkWriter *pWriter = NULL;
	DWORD nStreamIndex = 0;
	HRESULT hr;

	hr = MFCreateAttributes(&pSinkAttr, 2);
	if (SUCCEEDED(hr)) hr = IMFAttributes_SetUINT32(pSinkAttr, &MF_READWRITE_ENABLE_HARDWARE_TRANSFORMS, TRUE);
	if (SUCCEEDED(hr)) hr = MFCreateSinkWriterFromURL(wPath, NULL, pSinkAttr, &pWriter);
	if (FAILED(hr)) goto done;

	hr = MFCreateMediaType(&pOutType);
	if (SUCCEEDED(hr)) hr = IMFMediaType_SetGUID(pOutType, &MF_MT_MAJOR_TYPE, &MFMediaType_Video);
	if (SUCCEEDED(hr)) hr = IMFMediaType_SetGUID(pOutType, &MF_MT_SUBTYPE, &MFVideoFormat_H264);
	if (SUCCEEDED(hr)) hr = IMFMediaType_SetUINT32(pOutType, &MF_MT_AVG_BITRATE, (UINT32)iBitrate);
	if (SUCCEEDED(hr)) hr = IMFMediaType_SetUINT32(pOutType, &MF_MT_INTERLACE_MODE, MFVideoInterlace_Progressive);
	if (SUCCEEDED(hr)) hr = MFSetAttributeSize((IMFAttributes *)pOutType, &MF_MT_FRAME_SIZE, cx, cy);
	if (SUCCEEDED(hr)) hr = MFSetAttributeRatio((IMFAttributes *)pOutType, &MF_MT_FRAME_RATE, (UINT32)iFps, 1);
	if (SUCCEEDED(hr)) hr = MFSetAttributeRatio((IMFAttributes *)pOutType, &MF_MT_PIXEL_ASPECT_RATIO, 1, 1);
	if (FAILED(hr)) goto done;

	hr = IMFSinkWriter_AddStream(pWriter, pOutType, &nStreamIndex);
	if (FAILED(hr)) goto done;

	hr = MFCreateMediaType(&pInType);
	if (SUCCEEDED(hr)) hr = IMFMediaType_SetGUID(pInType, &MF_MT_MAJOR_TYPE, &MFMediaType_Video);
	if (SUCCEEDED(hr)) hr = IMFMediaType_SetGUID(pInType, &MF_MT_SUBTYPE, &MFVideoFormat_RGB32);
	if (SUCCEEDED(hr)) hr = IMFMediaType_SetUINT32(pInType, &MF_MT_INTERLACE_MODE, MFVideoInterlace_Progressive);
	if (SUCCEEDED(hr)) hr = MFSetAttributeSize((IMFAttributes *)pInType, &MF_MT_FRAME_SIZE, cx, cy);
	if (SUCCEEDED(hr)) hr = MFSetAttributeRatio((IMFAttributes *)pInType, &MF_MT_FRAME_RATE, (UINT32)iFps, 1);
	if (SUCCEEDED(hr)) hr = MFSetAttributeRatio((IMFAttributes *)pInType, &MF_MT_PIXEL_ASPECT_RATIO, 1, 1);
	if (FAILED(hr)) goto done;

	hr = IMFSinkWriter_SetInputMediaType(pWriter, nStreamIndex, pInType, NULL);
	if (FAILED(hr)) goto done;

	hr = IMFSinkWriter_BeginWriting(pWriter);
	if (FAILED(hr)) goto done;

	*ppWriter = pWriter;
	*pStreamIndex = nStreamIndex;
	pWriter = NULL; // ownership moved to *ppWriter

done:
	if (pInType != NULL) IMFMediaType_Release(pInType);
	if (pOutType != NULL) IMFMediaType_Release(pOutType);
	if (pSinkAttr != NULL) IMFAttributes_Release(pSinkAttr);
	if (pWriter != NULL) IMFSinkWriter_Release(pWriter);
	return hr;
}

// bgra_to_rgba copies a BGRA (Windows RGB32) buffer into a newly malloc'd, tightly packed RGBA
// buffer of the same size, swapping the B and R channels. The caller must free() the result.
static unsigned char *bgra_to_rgba(const unsigned char *pSrc, UINT32 iWidth, UINT32 iHeight, UINT32 iSrcStride,
                                    UINT32 *pOutStride) {
	UINT32 iDstStride = iWidth * 4;
	unsigned char *pDst = (unsigned char *)malloc((size_t)iDstStride * iHeight);
	if (pDst == NULL) return NULL;

	for (UINT32 y = 0; y < iHeight; y++) {
		const unsigned char *pRow = pSrc + (size_t)y * iSrcStride;
		unsigned char *pOut = pDst + (size_t)y * iDstStride;
		for (UINT32 x = 0; x < iWidth; x++) {
			pOut[x * 4 + 0] = pRow[x * 4 + 2]; // R
			pOut[x * 4 + 1] = pRow[x * 4 + 1]; // G
			pOut[x * 4 + 2] = pRow[x * 4 + 0]; // B
			pOut[x * 4 + 3] = pRow[x * 4 + 3]; // A
		}
	}
	*pOutStride = iDstStride;
	return pDst;
}

// bgra_downscale_to_rgba nearest-neighbor downscales a BGRA buffer to (iDstWidth, iDstHeight) RGBA
// (channels swapped too). The caller must free() the result.
static unsigned char *bgra_downscale_to_rgba(const unsigned char *pSrc, UINT32 iSrcWidth, UINT32 iSrcHeight,
                                              UINT32 iSrcStride, UINT32 iDstWidth, UINT32 iDstHeight,
                                              UINT32 *pOutStride) {
	UINT32 iDstStride = iDstWidth * 4;
	unsigned char *pDst = (unsigned char *)malloc((size_t)iDstStride * iDstHeight);
	if (pDst == NULL) return NULL;

	for (UINT32 y = 0; y < iDstHeight; y++) {
		UINT32 sy = (UINT32)((UINT64)y * iSrcHeight / iDstHeight);
		const unsigned char *pRow = pSrc + (size_t)sy * iSrcStride;
		unsigned char *pOut = pDst + (size_t)y * iDstStride;
		for (UINT32 x = 0; x < iDstWidth; x++) {
			UINT32 sx = (UINT32)((UINT64)x * iSrcWidth / iDstWidth);
			const unsigned char *pPixel = pRow + (size_t)sx * 4;
			pOut[x * 4 + 0] = pPixel[2];
			pOut[x * 4 + 1] = pPixel[1];
			pOut[x * 4 + 2] = pPixel[0];
			pOut[x * 4 + 3] = pPixel[3];
		}
	}
	*pOutStride = iDstStride;
	return pDst;
}

static unsigned __stdcall capture_thread_proc(void *pParam) {

	MFStream *pStream = (MFStream *)pParam;
	HRESULT hr;
	BOOL bMfInit = FALSE;

	IMFActivate *pActivate = NULL;
	IMFAttributes *pReaderAttr = NULL;
	IMFMediaType *pType = NULL;

	hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
	if (FAILED(hr) && hr != S_FALSE) {
		pStream->setupError = format_hresult("COM 初始化失敗", hr);
		pStream->setupResult = hr;
		SetEvent(pStream->hReadyEvent);
		return 0;
	}

	hr = MFStartup(MF_VERSION, MFSTARTUP_NOSOCKET);
	if (FAILED(hr)) {
		pStream->setupError = format_hresult("Media Foundation 啟動失敗", hr);
		pStream->setupResult = hr;
		SetEvent(pStream->hReadyEvent);
		CoUninitialize();
		return 0;
	}
	bMfInit = TRUE;

	hr = pick_device(pStream->sDevice, &pActivate);
	if (FAILED(hr)) {
		pStream->setupError = format_hresult("找不到攝影機,請確認已接上並允許攝影機權限", hr);
		goto setup_failed;
	}

	hr = IMFActivate_ActivateObject(pActivate, &IID_IMFMediaSource, (void **)&pStream->pSource);
	if (FAILED(hr)) {
		pStream->setupError = format_hresult("無法開啟攝影機裝置", hr);
		goto setup_failed;
	}

	hr = MFCreateAttributes(&pReaderAttr, 1);
	if (SUCCEEDED(hr)) hr = IMFAttributes_SetUINT32(pReaderAttr, &MF_SOURCE_READER_ENABLE_VIDEO_PROCESSING, TRUE);
	if (SUCCEEDED(hr)) hr = MFCreateSourceReaderFromMediaSource(pStream->pSource, pReaderAttr, &pStream->pReader);
	if (FAILED(hr)) {
		pStream->setupError = format_hresult("無法建立攝影機讀取器", hr);
		goto setup_failed;
	}

	hr = choose_media_type(pStream->pReader, pStream->width, pStream->height, &pType);
	if (FAILED(hr)) {
		pStream->setupError = format_hresult("攝影機不支援指定的解析度", hr);
		goto setup_failed;
	}

	hr = IMFSourceReader_SetCurrentMediaType(pStream->pReader, (DWORD)MF_SOURCE_READER_FIRST_VIDEO_STREAM, NULL, pType);
	if (FAILED(hr)) {
		pStream->setupError = format_hresult("攝影機不支援 RGB32 輸出", hr);
		goto setup_failed;
	}
	IMFMediaType_Release(pType);
	pType = NULL;

	hr = IMFSourceReader_GetCurrentMediaType(pStream->pReader, (DWORD)MF_SOURCE_READER_FIRST_VIDEO_STREAM, &pType);
	if (FAILED(hr)) {
		pStream->setupError = format_hresult("讀取攝影機格式失敗", hr);
		goto setup_failed;
	}

	{
		UINT32 cx = 0, cy = 0;
		hr = MFGetAttributeSize((IMFAttributes *)pType, &MF_MT_FRAME_SIZE, &cx, &cy);
		if (FAILED(hr) || cx == 0 || cy == 0) {
			pStream->setupError = format_hresult("攝影機畫面大小異常", hr);
			goto setup_failed;
		}

		LONG lStride = 0;
		if (FAILED(IMFMediaType_GetUINT32(pType, &MF_MT_DEFAULT_STRIDE, (UINT32 *)&lStride)) || lStride == 0) {
			lStride = (LONG)cx * 4;
		}
		if (lStride < 0) lStride = -lStride; // bottom-up DIB; see the note at the top of this file

		pStream->frameWidth = cx;
		pStream->frameHeight = cy;
		pStream->frameStride = (UINT32)lStride;

		if (pStream->wRecordPath != NULL) {
			hr = setup_sink_writer(pStream->wRecordPath, cx, cy, pStream->fps, pStream->bitrate, &pStream->pWriter,
			                        &pStream->writerStreamIndex);
			if (FAILED(hr)) {
				pStream->setupError = format_hresult("無法建立錄影檔", hr);
				goto setup_failed;
			}
		}
	}

	pStream->previewIntervalMs = (pStream->previewFps > 0) ? (1000.0 / pStream->previewFps) : 0.0;
	pStream->snapshotIntervalMs = (pStream->snapshotSeconds > 0) ? (pStream->snapshotSeconds * 1000.0) : 0.0;
	pStream->frameDuration100ns = (pStream->fps > 0) ? (10000000LL / pStream->fps) : 333333LL;

	IMFMediaType_Release(pType);
	pType = NULL;
	IMFAttributes_Release(pReaderAttr);
	pReaderAttr = NULL;
	IMFActivate_Release(pActivate);
	pActivate = NULL;

	pStream->setupResult = S_OK;
	SetEvent(pStream->hReadyEvent);

	// ---- capture loop ----
	{
		LONGLONG llNextTimestamp = 0;
		double dNextPreviewMs = 0.0;
		double dNextSnapshotMs = 0.0; // 0 => first snapshot right away, matching Options.SnapshotEvery's doc
		LARGE_INTEGER liFreq, liStart, liNow;
		QueryPerformanceFrequency(&liFreq);
		QueryPerformanceCounter(&liStart);

		for (;;) {
			if (pStream->stopFlag != 0) break;

			DWORD dwStreamFlags = 0;
			LONGLONG llTimestamp = 0;
			IMFSample *pSample = NULL;
			hr = IMFSourceReader_ReadSample(pStream->pReader, (DWORD)MF_SOURCE_READER_FIRST_VIDEO_STREAM, 0, NULL,
			                                 &dwStreamFlags, &llTimestamp, &pSample);
			if (FAILED(hr)) {
				char *cMsg = format_hresult("讀取攝影機畫面失敗", hr);
				if (cMsg != NULL) {
					goCameraError(pStream->handle, cMsg);
					free(cMsg);
				}
				break;
			}
			if (dwStreamFlags & MF_SOURCE_READERF_ENDOFSTREAM) {
				if (pSample != NULL) IMFSample_Release(pSample);
				goCameraError(pStream->handle, "攝影機已停止提供畫面");
				break;
			}
			if (pSample == NULL) continue; // gap, try again

			QueryPerformanceCounter(&liNow);
			double dElapsedMs = (double)(liNow.QuadPart - liStart.QuadPart) * 1000.0 / (double)liFreq.QuadPart;

			IMFMediaBuffer *pBuffer = NULL;
			if (SUCCEEDED(IMFSample_ConvertToContiguousBuffer(pSample, &pBuffer))) {
				BYTE *pData = NULL;
				DWORD cbMax = 0, cbCurrent = 0;
				if (SUCCEEDED(IMFMediaBuffer_Lock(pBuffer, &pData, &cbMax, &cbCurrent))) {

					if (pStream->previewWidth > 0 && pStream->previewHeight > 0 && dElapsedMs >= dNextPreviewMs) {
						UINT32 iOutStride = 0;
						unsigned char *pPreview =
						    bgra_downscale_to_rgba(pData, pStream->frameWidth, pStream->frameHeight, pStream->frameStride,
						                            (UINT32)pStream->previewWidth, (UINT32)pStream->previewHeight, &iOutStride);
						if (pPreview != NULL) {
							goCameraFrame(pStream->handle, MF_FRAME_PREVIEW, pPreview, pStream->previewWidth,
							              pStream->previewHeight, (int)iOutStride);
							free(pPreview);
						}
						dNextPreviewMs = dElapsedMs + pStream->previewIntervalMs;
					}

					if (pStream->snapshotIntervalMs > 0 && dElapsedMs >= dNextSnapshotMs) {
						UINT32 iOutStride = 0;
						unsigned char *pFull = bgra_to_rgba(pData, pStream->frameWidth, pStream->frameHeight,
						                                     pStream->frameStride, &iOutStride);
						if (pFull != NULL) {
							goCameraFrame(pStream->handle, MF_FRAME_SNAPSHOT, pFull, (int)pStream->frameWidth,
							              (int)pStream->frameHeight, (int)iOutStride);
							free(pFull);
						}
						dNextSnapshotMs = dElapsedMs + pStream->snapshotIntervalMs;
					}

					IMFMediaBuffer_Unlock(pBuffer);
				}
				IMFMediaBuffer_Release(pBuffer);
			}

			if (pStream->pWriter != NULL) {
				IMFSample_SetSampleTime(pSample, llNextTimestamp);
				IMFSample_SetSampleDuration(pSample, pStream->frameDuration100ns);
				llNextTimestamp += pStream->frameDuration100ns;
				IMFSinkWriter_WriteSample(pStream->pWriter, pStream->writerStreamIndex, pSample);
			}

			IMFSample_Release(pSample);
		}
	}
	goto cleanup;

setup_failed:
	if (pType != NULL) IMFMediaType_Release(pType);
	if (pReaderAttr != NULL) IMFAttributes_Release(pReaderAttr);
	if (pActivate != NULL) IMFActivate_Release(pActivate);
	pStream->setupResult = hr;
	SetEvent(pStream->hReadyEvent);

cleanup:
	if (pStream->pWriter != NULL) {
		IMFSinkWriter_Finalize(pStream->pWriter);
		IMFSinkWriter_Release(pStream->pWriter);
		pStream->pWriter = NULL;
	}
	if (pStream->pReader != NULL) {
		IMFSourceReader_Release(pStream->pReader);
		pStream->pReader = NULL;
	}
	if (pStream->pSource != NULL) {
		IMFMediaSource_Shutdown(pStream->pSource);
		IMFMediaSource_Release(pStream->pSource);
		pStream->pSource = NULL;
	}
	if (bMfInit) MFShutdown();
	CoUninitialize();
	return 0;
}

void *mf_stream_start(const char *cDevice, int iWidth, int iHeight, int iFps, int iBitrate, int iPreviewWidth,
                       int iPreviewHeight, int iPreviewFps, const char *cRecordPath, double dSnapshotSeconds,
                       uintptr_t iHandle, char **oError) {

	MFStream *pStream = (MFStream *)calloc(1, sizeof(MFStream));
	if (pStream == NULL) {
		*oError = format_hresult("記憶體不足", E_OUTOFMEMORY);
		return NULL;
	}

	pStream->handle = iHandle;
	pStream->width = iWidth;
	pStream->height = iHeight;
	pStream->fps = iFps;
	pStream->bitrate = iBitrate;
	pStream->previewWidth = iPreviewWidth;
	pStream->previewHeight = iPreviewHeight;
	pStream->previewFps = iPreviewFps;
	pStream->snapshotSeconds = dSnapshotSeconds;
	pStream->sDevice = (cDevice != NULL) ? _strdup(cDevice) : NULL;
	pStream->wRecordPath = (cRecordPath != NULL && cRecordPath[0] != '\0') ? utf8_to_wide(cRecordPath) : NULL;

	pStream->hReadyEvent = CreateEventW(NULL, TRUE, FALSE, NULL);
	if (pStream->hReadyEvent == NULL) {
		*oError = format_hresult("無法建立同步事件", HRESULT_FROM_WIN32(GetLastError()));
		free(pStream->sDevice);
		free(pStream->wRecordPath);
		free(pStream);
		return NULL;
	}

	pStream->hThread = (HANDLE)_beginthreadex(NULL, 0, capture_thread_proc, pStream, 0, NULL);
	if (pStream->hThread == NULL) {
		*oError = format_hresult("無法啟動擷取執行緒", HRESULT_FROM_WIN32(GetLastError()));
		CloseHandle(pStream->hReadyEvent);
		free(pStream->sDevice);
		free(pStream->wRecordPath);
		free(pStream);
		return NULL;
	}

	WaitForSingleObject(pStream->hReadyEvent, INFINITE);

	if (FAILED(pStream->setupResult)) {
		*oError = pStream->setupError; // ownership moves to the caller, who must free() it
		WaitForSingleObject(pStream->hThread, INFINITE);
		CloseHandle(pStream->hThread);
		CloseHandle(pStream->hReadyEvent);
		free(pStream->sDevice);
		free(pStream->wRecordPath);
		free(pStream);
		return NULL;
	}

	return (void *)pStream;
}

void mf_stream_stop(void *pStreamPtr) {

	if (pStreamPtr == NULL) return;
	MFStream *pStream = (MFStream *)pStreamPtr;

	InterlockedExchange(&pStream->stopFlag, 1);
	WaitForSingleObject(pStream->hThread, INFINITE);

	CloseHandle(pStream->hThread);
	CloseHandle(pStream->hReadyEvent);
	free(pStream->sDevice);
	free(pStream->wRecordPath);
	free(pStream->setupError);
	free(pStream);
}
