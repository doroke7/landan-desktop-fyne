//go:build windows && !cgo

// Windows Media Foundation backend for pkg/camera, without cgo.
//
// This is the same technique as stream_windows.go / mf_stream_windows.c - one Media Foundation
// source reader stream feeds a small RGBA preview, an H.264 mp4 via IMFSinkWriter, and periodic
// full-size RGBA snapshots - but every COM call is made by hand: COM interfaces are just vtables
// (arrays of function pointers) in memory, so a method call is "read the vtable pointer, read the
// function pointer at its index, call it via syscall.SyscallN with the object as the first arg".
// See vtblCall below. No cgo and no C compiler are needed, so `CGO_ENABLED=0 GOOS=windows go build`
// works from any OS.
//
// NOTE (written without a Windows machine to build/run against - verify on real hardware):
//   - The GUIDs and vtable method indices below are transcribed from the Windows SDK headers
//     (mfapi.h, mfidl.h, mfobjects.h, mfreadwrite.h) from memory, not compiled against them. A wrong
//     index calls the wrong method through the vtable (crash or silent nonsense); a wrong GUID makes
//     an attribute get/set silently fail. If something crashes or behaves oddly, check these first
//     against the actual SDK headers before looking elsewhere.
//   - MFEnumDeviceSources's DLL is genuinely ambiguous across SDK docs (mf.dll vs mfplat.dll); this
//     file tries both, see mfEnumDeviceSources.
//   - Frame rate/stride notes from mf_stream_windows.c apply here unchanged: the source reader runs
//     at the camera's native rate for the chosen resolution, and a bottom-up (negative) stride is
//     only sign-fixed, not flipped.
package camera

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

// ShowPreviewOverlay is only available on macOS (AVCaptureVideoPreviewLayer is an AVFoundation
// type). On this backend it is a no-op.
func ShowPreviewOverlay(fX, fY, fWidth, fHeight float32) {}

// HidePreviewOverlay is only available on macOS. On this backend it is a no-op.
func HidePreviewOverlay() {}

// ---- GUID plumbing ------------------------------------------------------------------------------

// guid matches the in-memory layout of a Windows GUID/IID/CLSID.
type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	iidIMFMediaSource = guid{0x279A808D, 0xAEC7, 0x40C8, [8]byte{0x9C, 0x6B, 0xA6, 0xB4, 0x92, 0xC7, 0x8A, 0x66}}

	mfDevsourceAttributeSourceType       = guid{0x58F0AAD8, 0x22BF, 0x4F8A, [8]byte{0xBB, 0x3D, 0xD2, 0xC4, 0x97, 0x8C, 0x6E, 0x2F}}
	mfDevsourceAttributeSourceTypeVidcap = guid{0x8AC3587A, 0x4AE7, 0x42D8, [8]byte{0x99, 0xE0, 0x0A, 0x60, 0x13, 0xEE, 0xF9, 0x0F}}
	mfDevsourceAttributeFriendlyName     = guid{0x60D0E559, 0x52F8, 0x4FA2, [8]byte{0xBB, 0xCE, 0xAC, 0xDB, 0x34, 0xA8, 0xEC, 0x01}}
	mfSourceReaderEnableVideoProcessing  = guid{0xFB394F3D, 0x4CEA, 0x4BA3, [8]byte{0x99, 0x68, 0xB3, 0x52, 0xC4, 0x0B, 0x0D, 0x4B}}
	mfReadwriteEnableHardwareTransforms  = guid{0xA634A91C, 0x822B, 0x41B9, [8]byte{0xA4, 0x94, 0x4D, 0xE4, 0x64, 0x36, 0x12, 0xB0}}

	mfMTMajorType        = guid{0x48EBA18E, 0xF8C9, 0x4687, [8]byte{0xBF, 0x11, 0x0A, 0x74, 0xC9, 0xF9, 0x6A, 0x8F}}
	mfMTSubtype          = guid{0xF7E34C9A, 0x42E8, 0x4714, [8]byte{0xB7, 0x4B, 0xCB, 0x29, 0xD7, 0x2C, 0x35, 0xE5}}
	mfMTFrameSize        = guid{0x1652C33D, 0xD6B2, 0x4012, [8]byte{0xB8, 0x34, 0x72, 0x03, 0x08, 0x49, 0xA3, 0x7D}}
	mfMTFrameRate        = guid{0xC459A2E8, 0x3D2C, 0x4E44, [8]byte{0xB1, 0x32, 0xFE, 0xE5, 0x15, 0x6C, 0x7B, 0xB0}}
	mfMTPixelAspectRatio = guid{0xC6376A1E, 0x8D0A, 0x4027, [8]byte{0xBE, 0x45, 0x6D, 0x9A, 0x0A, 0xD3, 0x9B, 0xB6}}
	mfMTInterlaceMode    = guid{0xE2724BB8, 0xE676, 0x4806, [8]byte{0xB4, 0xB2, 0xA8, 0xD6, 0xEF, 0xB4, 0x4C, 0xCD}}
	mfMTDefaultStride    = guid{0x644B4E48, 0x1E02, 0x4516, [8]byte{0xB0, 0xEB, 0xC0, 0x1C, 0xA9, 0xD4, 0x9A, 0xC6}}
	mfMTAvgBitrate       = guid{0x20332624, 0xFB0D, 0x4D9E, [8]byte{0xBD, 0x0D, 0xCB, 0xF6, 0x78, 0x6C, 0x10, 0x2E}}

	mfMediaTypeVideo   = guid{0x73646976, 0x0000, 0x0010, [8]byte{0x80, 0x00, 0x00, 0xAA, 0x00, 0x38, 0x9B, 0x71}}
	mfVideoFormatRGB32 = guid{0x00000016, 0x0000, 0x0010, [8]byte{0x80, 0x00, 0x00, 0xAA, 0x00, 0x38, 0x9B, 0x71}}
	mfVideoFormatH264  = guid{0x34363248, 0x0000, 0x0010, [8]byte{0x80, 0x00, 0x00, 0xAA, 0x00, 0x38, 0x9B, 0x71}}
)

const (
	mfVideoInterlaceProgressive = 2

	mfSourceReaderFirstVideoStream = 0xFFFFFFFC // (DWORD)-4
	mfSourceReaderfEndOfStream     = 0x2
	mfENoMoreTypes                 = 0xC00D36B9 // HRESULT MF_E_NO_MORE_TYPES

	mfVersion           = 0x00020070
	mfStartupNosocket   = 0x1
	coinitMultithreaded = 0
)

// ---- vtable-based COM calls -----------------------------------------------------------------

// Every COM interface starts with IUnknown: QueryInterface, AddRef, Release at indices 0-2.
const (
	idxAddRef  = 1
	idxRelease = 2

	// IMFAttributes (also the base of IMFActivate, IMFMediaType, IMFSample).
	idxAttrGetUINT32          = 7
	idxAttrGetUINT64          = 8
	idxAttrGetGUID            = 10
	idxAttrGetAllocatedString = 13
	idxAttrSetUINT32          = 21
	idxAttrSetUINT64          = 22
	idxAttrSetGUID            = 24
	idxAttrCopyAllItems       = 32

	idxActivateActivateObject = 33 // IMFActivate, right after IMFAttributes' 30 methods (3..32)

	idxSourceShutdown = 12 // IMFMediaSource, after IMFMediaEventGenerator's 4 methods (3..6)

	idxReaderGetNativeMediaType  = 5 // IMFSourceReader
	idxReaderGetCurrentMediaType = 6
	idxReaderSetCurrentMediaType = 7
	idxReaderReadSample          = 9

	idxWriterAddStream         = 3 // IMFSinkWriter
	idxWriterSetInputMediaType = 4
	idxWriterBeginWriting      = 5
	idxWriterWriteSample       = 6
	idxWriterFinalize          = 11

	idxBufferLock   = 3 // IMFMediaBuffer
	idxBufferUnlock = 4

	idxSampleSetSampleTime             = 36 // IMFSample, after IMFAttributes' 30 methods (3..32) plus 3 (33..35)
	idxSampleSetSampleDuration         = 38
	idxSampleConvertToContiguousBuffer = 41
)

// vtblCall invokes the method at vtable index on a COM object, with obj passed as the implicit
// "this" first argument - the same ABI cgo's COBJMACROS wrappers (IMFxxx_Method(p, ...)) use.
func vtblCall(obj unsafe.Pointer, index uintptr, a ...uintptr) uintptr {
	vtbl := *(*uintptr)(obj)
	fn := *(*uintptr)(unsafe.Pointer(vtbl + index*unsafe.Sizeof(uintptr(0))))
	aArgs := make([]uintptr, 0, len(a)+1)
	aArgs = append(aArgs, uintptr(obj))
	aArgs = append(aArgs, a...)
	iR1, _, _ := syscall.SyscallN(fn, aArgs...)
	return iR1
}

func comAddRef(pObj unsafe.Pointer) {
	if pObj != nil {
		vtblCall(pObj, idxAddRef)
	}
}

func comRelease(pObj unsafe.Pointer) {
	if pObj != nil {
		vtblCall(pObj, idxRelease)
	}
}

func failed(iHR uint32) bool { return int32(iHR) < 0 }

func hresultErr(sPrefix string, iHR uint32) error {
	return fmt.Errorf("%s (hr=0x%08X)", sPrefix, iHR)
}

func packUint32Pair(iHi, iLo uint32) uint64           { return uint64(iHi)<<32 | uint64(iLo) }
func unpackUint32Pair(iValue uint64) (uint32, uint32) { return uint32(iValue >> 32), uint32(iValue) }

func attrGetUINT32(pObj unsafe.Pointer, pKey *guid) (uint32, uint32) {
	var iValue uint32
	iHR := vtblCall(pObj, idxAttrGetUINT32, uintptr(unsafe.Pointer(pKey)), uintptr(unsafe.Pointer(&iValue)))
	return iValue, uint32(iHR)
}

func attrSetUINT32(pObj unsafe.Pointer, pKey *guid, iValue uint32) uint32 {
	return uint32(vtblCall(pObj, idxAttrSetUINT32, uintptr(unsafe.Pointer(pKey)), uintptr(iValue)))
}

func attrGetUINT64(pObj unsafe.Pointer, pKey *guid) (uint64, uint32) {
	var iValue uint64
	iHR := vtblCall(pObj, idxAttrGetUINT64, uintptr(unsafe.Pointer(pKey)), uintptr(unsafe.Pointer(&iValue)))
	return iValue, uint32(iHR)
}

func attrSetUINT64(pObj unsafe.Pointer, pKey *guid, iValue uint64) uint32 {
	return uint32(vtblCall(pObj, idxAttrSetUINT64, uintptr(unsafe.Pointer(pKey)), uintptr(iValue)))
}

func attrSetGUID(pObj unsafe.Pointer, pKey *guid, pValue *guid) uint32 {
	return uint32(vtblCall(pObj, idxAttrSetGUID, uintptr(unsafe.Pointer(pKey)), uintptr(unsafe.Pointer(pValue))))
}

func attrCopyAllItems(pSrc, pDst unsafe.Pointer) uint32 {
	return uint32(vtblCall(pSrc, idxAttrCopyAllItems, uintptr(pDst)))
}

// attrGetAllocatedString returns "" (with a non-zero hr) on failure. On success the returned wide
// string (caller must CoTaskMemFree) is freed before returning.
func attrGetAllocatedString(pObj unsafe.Pointer, pKey *guid) (string, uint32) {
	var pStr *uint16
	var iLen uint32
	iHR := vtblCall(pObj, idxAttrGetAllocatedString,
		uintptr(unsafe.Pointer(pKey)), uintptr(unsafe.Pointer(&pStr)), uintptr(unsafe.Pointer(&iLen)))
	if failed(uint32(iHR)) || pStr == nil {
		return "", uint32(iHR)
	}
	defer coTaskMemFree(unsafe.Pointer(pStr))
	return string(utf16.Decode(unsafe.Slice(pStr, iLen))), 0
}

func activateObject(pActivate unsafe.Pointer, pIID *guid) (unsafe.Pointer, uint32) {
	var pOut unsafe.Pointer
	iHR := vtblCall(pActivate, idxActivateActivateObject, uintptr(unsafe.Pointer(pIID)), uintptr(unsafe.Pointer(&pOut)))
	return pOut, uint32(iHR)
}

func getNativeMediaType(pReader unsafe.Pointer, iStreamIndex, iTypeIndex uint32) (unsafe.Pointer, uint32) {
	var pType unsafe.Pointer
	iHR := vtblCall(pReader, idxReaderGetNativeMediaType, uintptr(iStreamIndex), uintptr(iTypeIndex), uintptr(unsafe.Pointer(&pType)))
	return pType, uint32(iHR)
}

func getCurrentMediaType(pReader unsafe.Pointer, iStreamIndex uint32) (unsafe.Pointer, uint32) {
	var pType unsafe.Pointer
	iHR := vtblCall(pReader, idxReaderGetCurrentMediaType, uintptr(iStreamIndex), uintptr(unsafe.Pointer(&pType)))
	return pType, uint32(iHR)
}

func setCurrentMediaType(pReader unsafe.Pointer, iStreamIndex uint32, pType unsafe.Pointer) uint32 {
	return uint32(vtblCall(pReader, idxReaderSetCurrentMediaType, uintptr(iStreamIndex), 0, uintptr(pType)))
}

func readSample(pReader unsafe.Pointer, iStreamIndex uint32) (iStreamFlags uint32, pSample unsafe.Pointer, iHR uint32) {
	var iFlags uint32
	var iTimestamp int64
	var pOut unsafe.Pointer
	r := vtblCall(pReader, idxReaderReadSample,
		uintptr(iStreamIndex), 0, 0,
		uintptr(unsafe.Pointer(&iFlags)), uintptr(unsafe.Pointer(&iTimestamp)), uintptr(unsafe.Pointer(&pOut)))
	return iFlags, pOut, uint32(r)
}

func bufferLock(pBuffer unsafe.Pointer) (unsafe.Pointer, uint32) {
	var pData unsafe.Pointer
	var iMax, iCur uint32
	iHR := vtblCall(pBuffer, idxBufferLock, uintptr(unsafe.Pointer(&pData)), uintptr(unsafe.Pointer(&iMax)), uintptr(unsafe.Pointer(&iCur)))
	return pData, uint32(iHR)
}

func bufferUnlock(pBuffer unsafe.Pointer) { vtblCall(pBuffer, idxBufferUnlock) }

func sampleConvertToContiguousBuffer(pSample unsafe.Pointer) (unsafe.Pointer, uint32) {
	var pBuffer unsafe.Pointer
	iHR := vtblCall(pSample, idxSampleConvertToContiguousBuffer, uintptr(unsafe.Pointer(&pBuffer)))
	return pBuffer, uint32(iHR)
}

func sampleSetSampleTime(pSample unsafe.Pointer, iTime int64) {
	vtblCall(pSample, idxSampleSetSampleTime, uintptr(iTime))
}

func sampleSetSampleDuration(pSample unsafe.Pointer, iDuration int64) {
	vtblCall(pSample, idxSampleSetSampleDuration, uintptr(iDuration))
}

func writerAddStream(pWriter unsafe.Pointer, pType unsafe.Pointer) (uint32, uint32) {
	var iIndex uint32
	iHR := vtblCall(pWriter, idxWriterAddStream, uintptr(pType), uintptr(unsafe.Pointer(&iIndex)))
	return iIndex, uint32(iHR)
}

func writerSetInputMediaType(pWriter unsafe.Pointer, iStreamIndex uint32, pType unsafe.Pointer) uint32 {
	return uint32(vtblCall(pWriter, idxWriterSetInputMediaType, uintptr(iStreamIndex), uintptr(pType), 0))
}

func writerBeginWriting(pWriter unsafe.Pointer) uint32 {
	return uint32(vtblCall(pWriter, idxWriterBeginWriting))
}

func writerWriteSample(pWriter unsafe.Pointer, iStreamIndex uint32, pSample unsafe.Pointer) uint32 {
	return uint32(vtblCall(pWriter, idxWriterWriteSample, uintptr(iStreamIndex), uintptr(pSample)))
}

func writerFinalize(pWriter unsafe.Pointer) uint32 {
	return uint32(vtblCall(pWriter, idxWriterFinalize))
}

// ---- flat (non-vtable) DLL exports ------------------------------------------------------------

var (
	dllOle32       = syscall.NewLazyDLL("ole32.dll")
	dllMFPlat      = syscall.NewLazyDLL("mfplat.dll")
	dllMFReadWrite = syscall.NewLazyDLL("mfreadwrite.dll")
	dllMF          = syscall.NewLazyDLL("mf.dll")

	procCoInitializeEx = dllOle32.NewProc("CoInitializeEx")
	procCoUninitialize = dllOle32.NewProc("CoUninitialize")
	procCoTaskMemFree  = dllOle32.NewProc("CoTaskMemFree")

	procMFStartup          = dllMFPlat.NewProc("MFStartup")
	procMFShutdown         = dllMFPlat.NewProc("MFShutdown")
	procMFCreateAttributes = dllMFPlat.NewProc("MFCreateAttributes")
	procMFCreateMediaType  = dllMFPlat.NewProc("MFCreateMediaType")

	// See the file-level NOTE: this export's actual DLL differs across SDK/OS versions, so both are tried.
	procMFEnumDeviceSourcesMF     = dllMF.NewProc("MFEnumDeviceSources")
	procMFEnumDeviceSourcesMFPlat = dllMFPlat.NewProc("MFEnumDeviceSources")

	procMFCreateSourceReaderFromMediaSource = dllMFReadWrite.NewProc("MFCreateSourceReaderFromMediaSource")
	procMFCreateSinkWriterFromURL           = dllMFReadWrite.NewProc("MFCreateSinkWriterFromURL")
)

// callProc calls a LazyProc without panicking if it can't be resolved (LazyProc.Call panics on a
// missing export; Find lets us turn that into a normal error instead).
func callProc(pProc *syscall.LazyProc, a ...uintptr) (uintptr, error) {
	if err := pProc.Find(); err != nil {
		return 0, err
	}
	r1, _, _ := pProc.Call(a...)
	return r1, nil
}

func coTaskMemFree(pMem unsafe.Pointer) { procCoTaskMemFree.Call(uintptr(pMem)) }

func coInitialize() uint32 {
	r, _, _ := procCoInitializeEx.Call(0, coinitMultithreaded)
	if uint32(r) == 1 { // S_FALSE: already initialized on this thread, treated as success like the cgo version
		return 0
	}
	return uint32(r)
}

func coUninitialize() { procCoUninitialize.Call() }

func mfStartup() uint32 {
	r, _, _ := procMFStartup.Call(mfVersion, mfStartupNosocket)
	return uint32(r)
}

func mfShutdown() { procMFShutdown.Call() }

func mfCreateAttributes(iInitialSize uint32) (unsafe.Pointer, error) {
	var pOut unsafe.Pointer
	r, _, _ := procMFCreateAttributes.Call(uintptr(unsafe.Pointer(&pOut)), uintptr(iInitialSize))
	if iHR := uint32(r); failed(iHR) {
		return nil, hresultErr("MFCreateAttributes", iHR)
	}
	return pOut, nil
}

func mfCreateMediaType() (unsafe.Pointer, error) {
	var pOut unsafe.Pointer
	r, _, _ := procMFCreateMediaType.Call(uintptr(unsafe.Pointer(&pOut)))
	if iHR := uint32(r); failed(iHR) {
		return nil, hresultErr("MFCreateMediaType", iHR)
	}
	return pOut, nil
}

// mfEnumDeviceSources returns the caller-owned (AddRef'd once each, by this function) list of video
// capture device activators.
func mfEnumDeviceSources(pAttr unsafe.Pointer) ([]unsafe.Pointer, error) {
	var pDevices unsafe.Pointer
	var iCount uint32
	r, err := callProc(procMFEnumDeviceSourcesMF, uintptr(pAttr), uintptr(unsafe.Pointer(&pDevices)), uintptr(unsafe.Pointer(&iCount)))
	if err != nil {
		r, err = callProc(procMFEnumDeviceSourcesMFPlat, uintptr(pAttr), uintptr(unsafe.Pointer(&pDevices)), uintptr(unsafe.Pointer(&iCount)))
	}
	if err != nil {
		return nil, fmt.Errorf("找不到 MFEnumDeviceSources: %w", err)
	}
	if iHR := uint32(r); failed(iHR) {
		return nil, hresultErr("MFEnumDeviceSources", iHR)
	}
	if iCount == 0 {
		return nil, nil
	}
	defer coTaskMemFree(pDevices)

	aRaw := unsafe.Slice((*unsafe.Pointer)(pDevices), iCount)
	aOut := make([]unsafe.Pointer, iCount)
	copy(aOut, aRaw)
	return aOut, nil
}

// ---- device selection and media type negotiation -----------------------------------------------

// pickDevice selects a video capture device: "" / "default" picks the first one, a plain integer
// picks by zero-based index, anything else matches (case-insensitively) against a substring of the
// device's friendly name - same convention as Options.Device documents for the other backends.
func pickDevice(sDevice string) (unsafe.Pointer, error) {

	pAttr, err := mfCreateAttributes(1)
	if err != nil {
		return nil, err
	}
	defer comRelease(pAttr)
	if iHR := attrSetGUID(pAttr, &mfDevsourceAttributeSourceType, &mfDevsourceAttributeSourceTypeVidcap); failed(iHR) {
		return nil, hresultErr("SetGUID(SOURCE_TYPE)", iHR)
	}

	aDevices, err := mfEnumDeviceSources(pAttr)
	if err != nil {
		return nil, err
	}
	if len(aDevices) == 0 {
		return nil, errors.New("找不到攝影機裝置")
	}
	defer func() {
		for _, pDevice := range aDevices {
			comRelease(pDevice)
		}
	}()

	var pChosen unsafe.Pointer
	sTrim := strings.TrimSpace(sDevice)
	switch {
	case sTrim == "" || strings.EqualFold(sTrim, "default"):
		pChosen = aDevices[0]
	default:
		if iIndex, err := strconv.Atoi(sTrim); err == nil && iIndex >= 0 && iIndex < len(aDevices) {
			pChosen = aDevices[iIndex]
		} else {
			sNeedle := strings.ToLower(sTrim)
			for _, pDevice := range aDevices {
				sName, iHR := attrGetAllocatedString(pDevice, &mfDevsourceAttributeFriendlyName)
				if !failed(iHR) && strings.Contains(strings.ToLower(sName), sNeedle) {
					pChosen = pDevice
					break
				}
			}
		}
	}
	if pChosen == nil {
		return nil, fmt.Errorf("找不到符合的攝影機裝置: %q", sDevice)
	}
	comAddRef(pChosen) // ownership moves to the caller; the deferred loop above releases our own ref
	return pChosen, nil
}

// chooseMediaType picks the native type on the reader's first video stream whose frame size is
// closest to (iWidth, iHeight), and returns a copy of it with the subtype switched to RGB32 - the
// video processor (MF_SOURCE_READER_ENABLE_VIDEO_PROCESSING) then converts the camera's native
// format to RGB32 at that same frame size.
func chooseMediaType(pReader unsafe.Pointer, iWidth, iHeight int) (unsafe.Pointer, error) {

	var pBest unsafe.Pointer
	iBestDiff := int64(-1)
	defer comRelease(pBest)

	for i := uint32(0); ; i++ {
		pType, iHR := getNativeMediaType(pReader, mfSourceReaderFirstVideoStream, i)
		if iHR == mfENoMoreTypes {
			break
		}
		if failed(iHR) {
			return nil, hresultErr("GetNativeMediaType", iHR)
		}

		if iSize, sHR := attrGetUINT64(pType, &mfMTFrameSize); !failed(sHR) {
			iCX, iCY := unpackUint32Pair(iSize)
			iDX, iDY := int64(iCX)-int64(iWidth), int64(iCY)-int64(iHeight)
			iDiff := iDX*iDX + iDY*iDY
			if iBestDiff < 0 || iDiff < iBestDiff {
				comRelease(pBest)
				pBest = pType
				comAddRef(pBest)
				iBestDiff = iDiff
			}
		}
		comRelease(pType)
	}
	if pBest == nil {
		return nil, errors.New("攝影機不支援指定的解析度")
	}

	pNewType, err := mfCreateMediaType()
	if err != nil {
		return nil, err
	}
	if iHR := attrCopyAllItems(pBest, pNewType); failed(iHR) {
		comRelease(pNewType)
		return nil, hresultErr("CopyAllItems", iHR)
	}
	if iHR := attrSetGUID(pNewType, &mfMTSubtype, &mfVideoFormatRGB32); failed(iHR) {
		comRelease(pNewType)
		return nil, hresultErr("SetGUID(SUBTYPE)", iHR)
	}
	if iHR := attrSetUINT32(pNewType, &mfMTInterlaceMode, mfVideoInterlaceProgressive); failed(iHR) {
		comRelease(pNewType)
		return nil, hresultErr("SetUINT32(INTERLACE_MODE)", iHR)
	}
	return pNewType, nil
}

// setupSinkWriter opens an mp4 sink writer that encodes the RGB32 frames we feed it (cx x cy) into
// H.264 at iBitrate bits/second, iFPS frames/second.
func setupSinkWriter(sPath string, iCX, iCY uint32, iFPS, iBitrate int) (unsafe.Pointer, uint32, error) {

	pSinkAttr, err := mfCreateAttributes(2)
	if err != nil {
		return nil, 0, err
	}
	defer comRelease(pSinkAttr)
	if iHR := attrSetUINT32(pSinkAttr, &mfReadwriteEnableHardwareTransforms, 1); failed(iHR) {
		return nil, 0, hresultErr("SetUINT32(HW_TRANSFORMS)", iHR)
	}

	wPath, err := syscall.UTF16PtrFromString(sPath)
	if err != nil {
		return nil, 0, err
	}

	var pWriter unsafe.Pointer
	r, _, _ := procMFCreateSinkWriterFromURL.Call(uintptr(unsafe.Pointer(wPath)), 0, uintptr(pSinkAttr), uintptr(unsafe.Pointer(&pWriter)))
	if iHR := uint32(r); failed(iHR) {
		return nil, 0, hresultErr("MFCreateSinkWriterFromURL", iHR)
	}

	pOutType, err := mfCreateMediaType()
	if err != nil {
		comRelease(pWriter)
		return nil, 0, err
	}
	attrSetGUID(pOutType, &mfMTMajorType, &mfMediaTypeVideo)
	attrSetGUID(pOutType, &mfMTSubtype, &mfVideoFormatH264)
	attrSetUINT32(pOutType, &mfMTAvgBitrate, uint32(iBitrate))
	attrSetUINT32(pOutType, &mfMTInterlaceMode, mfVideoInterlaceProgressive)
	attrSetUINT64(pOutType, &mfMTFrameSize, packUint32Pair(iCX, iCY))
	attrSetUINT64(pOutType, &mfMTFrameRate, packUint32Pair(uint32(iFPS), 1))
	attrSetUINT64(pOutType, &mfMTPixelAspectRatio, packUint32Pair(1, 1))

	iStreamIndex, iHR := writerAddStream(pWriter, pOutType)
	if failed(iHR) {
		comRelease(pOutType)
		comRelease(pWriter)
		return nil, 0, hresultErr("AddStream", iHR)
	}

	pInType, err := mfCreateMediaType()
	if err != nil {
		comRelease(pOutType)
		comRelease(pWriter)
		return nil, 0, err
	}
	attrSetGUID(pInType, &mfMTMajorType, &mfMediaTypeVideo)
	attrSetGUID(pInType, &mfMTSubtype, &mfVideoFormatRGB32)
	attrSetUINT32(pInType, &mfMTInterlaceMode, mfVideoInterlaceProgressive)
	attrSetUINT64(pInType, &mfMTFrameSize, packUint32Pair(iCX, iCY))
	attrSetUINT64(pInType, &mfMTFrameRate, packUint32Pair(uint32(iFPS), 1))
	attrSetUINT64(pInType, &mfMTPixelAspectRatio, packUint32Pair(1, 1))

	iHR = writerSetInputMediaType(pWriter, iStreamIndex, pInType)
	comRelease(pInType)
	comRelease(pOutType)
	if failed(iHR) {
		comRelease(pWriter)
		return nil, 0, hresultErr("SetInputMediaType", iHR)
	}

	if iHR := writerBeginWriting(pWriter); failed(iHR) {
		comRelease(pWriter)
		return nil, 0, hresultErr("BeginWriting", iHR)
	}
	return pWriter, iStreamIndex, nil
}

// ---- pixel conversion and snapshots -------------------------------------------------------------

// lockedBufferToRGBA copies a locked BGRA (Windows RGB32) buffer into an image.RGBA, swapping the
// B and R channels. The buffer is only valid while the IMFMediaBuffer stays locked.
func lockedBufferToRGBA(pData unsafe.Pointer, iWidth, iHeight, iStride int) *image.RGBA {
	aSrc := unsafe.Slice((*byte)(pData), iStride*iHeight)
	oImage := image.NewRGBA(image.Rect(0, 0, iWidth, iHeight))
	for y := range iHeight {
		aRow := aSrc[y*iStride : y*iStride+iWidth*4]
		aOut := oImage.Pix[y*oImage.Stride : y*oImage.Stride+iWidth*4]
		for x := range iWidth {
			aOut[x*4+0] = aRow[x*4+2]
			aOut[x*4+1] = aRow[x*4+1]
			aOut[x*4+2] = aRow[x*4+0]
			aOut[x*4+3] = aRow[x*4+3]
		}
	}
	return oImage
}

// lockedBufferDownscaleToRGBA nearest-neighbor downscales a locked BGRA buffer to (iDstWidth,
// iDstHeight) RGBA (channels swapped too).
func lockedBufferDownscaleToRGBA(pData unsafe.Pointer, iSrcWidth, iSrcHeight, iSrcStride, iDstWidth, iDstHeight int) *image.RGBA {
	aSrc := unsafe.Slice((*byte)(pData), iSrcStride*iSrcHeight)
	oImage := image.NewRGBA(image.Rect(0, 0, iDstWidth, iDstHeight))
	for y := range iDstHeight {
		iSY := y * iSrcHeight / iDstHeight
		aRow := aSrc[iSY*iSrcStride:]
		aOut := oImage.Pix[y*oImage.Stride : y*oImage.Stride+iDstWidth*4]
		for x := range iDstWidth {
			iSX := x * iSrcWidth / iDstWidth
			aOut[x*4+0] = aRow[iSX*4+2]
			aOut[x*4+1] = aRow[iSX*4+1]
			aOut[x*4+2] = aRow[iSX*4+0]
			aOut[x*4+3] = aRow[iSX*4+3]
		}
	}
	return oImage
}

// saveSnapshot writes oImage into sDirectory, named by time.
func saveSnapshot(sDirectory string, oImage image.Image) {

	var oBuffer bytes.Buffer
	if err := jpeg.Encode(&oBuffer, oImage, &jpeg.Options{Quality: 92}); err != nil {
		log.Printf("截圖編碼失敗: %v", err)
		return
	}

	sPath := filepath.Join(sDirectory, "snapshot-"+time.Now().Format("20060102-150405")+".jpg")
	if err := os.WriteFile(sPath, oBuffer.Bytes(), 0o644); err != nil {
		log.Printf("截圖存檔失敗: %v", err)
	}
}

// ---- Stream ---------------------------------------------------------------------------------

// Stream reads the webcam through Media Foundation, called via raw COM vtable pointers (no cgo).
// One capture session feeds three things: a small live preview (fnFrame), an H.264 mp4 file, and
// periodic full-size JPEG snapshots. Cancelling oCtx stops the camera and finalizes the mp4.
//
// Everything - device enumeration through teardown - runs on one dedicated goroutine locked to its
// OS thread (runtime.LockOSThread), so the whole session stays on a single COM apartment (MTA), the
// same way mf_stream_windows.c's capture_thread_proc does with a dedicated Windows thread.
func Stream(oCtx context.Context, oOptions Options, fnFrame func(image.Image)) error {

	var iBitrate int
	if oOptions.RecordPath != "" {
		var err error
		if iBitrate, err = parseBitrate(oOptions.Bitrate); err != nil {
			return err
		}
	}

	var dSnapshotSeconds float64
	if oOptions.SnapshotDir != "" && oOptions.SnapshotEvery >= time.Second {
		dSnapshotSeconds = oOptions.SnapshotEvery.Seconds()
	}

	chReady := make(chan error, 1)
	chStop := make(chan struct{})
	chFailed := make(chan error, 1)
	chDone := make(chan struct{})

	go captureWorker(oOptions, iBitrate, dSnapshotSeconds, fnFrame, chReady, chStop, chFailed, chDone)

	if err := <-chReady; err != nil {
		return err
	}

	var errCamera error
	select {
	case <-oCtx.Done():
	case errCamera = <-chFailed:
	}

	// Signals the worker to stop and waits until it has torn everything down (mp4 finalized, COM
	// uninitialized); no callback runs after this, matching mf_stream_stop's contract.
	close(chStop)
	<-chDone

	if oCtx.Err() != nil {
		return nil
	}
	return errCamera
}

func captureWorker(oOptions Options, iBitrate int, dSnapshotSeconds float64, fnFrame func(image.Image),
	chReady chan<- error, chStop <-chan struct{}, chFailed chan<- error, chDone chan<- struct{}) {

	runtime.LockOSThread()
	defer close(chDone) // registered first so it runs last, after every deferred COM cleanup below

	if iHR := coInitialize(); failed(iHR) {
		chReady <- hresultErr("COM 初始化失敗", iHR)
		return
	}
	defer coUninitialize()

	if iHR := mfStartup(); failed(iHR) {
		chReady <- hresultErr("Media Foundation 啟動失敗", iHR)
		return
	}
	defer mfShutdown()

	pActivate, err := pickDevice(oOptions.Device)
	if err != nil {
		chReady <- fmt.Errorf("找不到攝影機,請確認已接上並允許攝影機權限: %w", err)
		return
	}
	pSource, iHR := activateObject(pActivate, &iidIMFMediaSource)
	comRelease(pActivate)
	if failed(iHR) {
		chReady <- hresultErr("無法開啟攝影機裝置", iHR)
		return
	}
	defer func() {
		vtblCall(pSource, idxSourceShutdown)
		comRelease(pSource)
	}()

	pReaderAttr, err := mfCreateAttributes(1)
	if err != nil {
		chReady <- err
		return
	}
	attrSetUINT32(pReaderAttr, &mfSourceReaderEnableVideoProcessing, 1)

	var pReader unsafe.Pointer
	r, _, _ := procMFCreateSourceReaderFromMediaSource.Call(uintptr(pSource), uintptr(pReaderAttr), uintptr(unsafe.Pointer(&pReader)))
	comRelease(pReaderAttr)
	if iHR := uint32(r); failed(iHR) {
		chReady <- hresultErr("無法建立攝影機讀取器", iHR)
		return
	}
	defer comRelease(pReader)

	pType, err := chooseMediaType(pReader, oOptions.Width, oOptions.Height)
	if err != nil {
		chReady <- err
		return
	}
	iHR = setCurrentMediaType(pReader, mfSourceReaderFirstVideoStream, pType)
	comRelease(pType)
	if failed(iHR) {
		chReady <- hresultErr("攝影機不支援 RGB32 輸出", iHR)
		return
	}

	pCurType, iHR := getCurrentMediaType(pReader, mfSourceReaderFirstVideoStream)
	if failed(iHR) {
		chReady <- hresultErr("讀取攝影機格式失敗", iHR)
		return
	}
	iSize, sHR := attrGetUINT64(pCurType, &mfMTFrameSize)
	iCX, iCY := unpackUint32Pair(iSize)
	if failed(sHR) || iCX == 0 || iCY == 0 {
		comRelease(pCurType)
		chReady <- errors.New("攝影機畫面大小異常")
		return
	}
	iStride, strideHR := attrGetUINT32(pCurType, &mfMTDefaultStride)
	if failed(strideHR) || iStride == 0 {
		iStride = iCX * 4
	} else if int32(iStride) < 0 {
		iStride = uint32(-int32(iStride)) // bottom-up DIB; see the file-level NOTE
	}
	comRelease(pCurType)

	var pWriter unsafe.Pointer
	var iWriterStreamIndex uint32
	if oOptions.RecordPath != "" {
		pWriter, iWriterStreamIndex, err = setupSinkWriter(oOptions.RecordPath, iCX, iCY, oOptions.Framerate, iBitrate)
		if err != nil {
			chReady <- fmt.Errorf("無法建立錄影檔: %w", err)
			return
		}
		defer func() {
			writerFinalize(pWriter)
			comRelease(pWriter)
		}()
	}

	dPreviewIntervalMs := 0.0
	if oOptions.PreviewFramerate > 0 {
		dPreviewIntervalMs = 1000.0 / float64(oOptions.PreviewFramerate)
	}
	dSnapshotIntervalMs := dSnapshotSeconds * 1000.0
	iFrameDuration100ns := int64(333333)
	if oOptions.Framerate > 0 {
		iFrameDuration100ns = 10000000 / int64(oOptions.Framerate)
	}

	chReady <- nil

	var iNextTimestamp int64
	dNextPreviewMs, dNextSnapshotMs := 0.0, 0.0 // 0 => first preview/snapshot right away
	tStart := time.Now()

	for {
		select {
		case <-chStop:
			return
		default:
		}

		iFlags, pSample, iHR := readSample(pReader, mfSourceReaderFirstVideoStream)
		if failed(iHR) {
			chFailed <- hresultErr("讀取攝影機畫面失敗", iHR)
			return
		}
		if iFlags&mfSourceReaderfEndOfStream != 0 {
			comRelease(pSample)
			chFailed <- errors.New("攝影機已停止提供畫面")
			return
		}
		if pSample == nil {
			continue // gap, try again
		}

		dElapsedMs := time.Since(tStart).Seconds() * 1000.0

		if pBuffer, bHR := sampleConvertToContiguousBuffer(pSample); !failed(bHR) {
			if pData, lHR := bufferLock(pBuffer); !failed(lHR) {

				if oOptions.PreviewWidth > 0 && oOptions.PreviewHeight > 0 && dElapsedMs >= dNextPreviewMs {
					oImage := lockedBufferDownscaleToRGBA(pData, int(iCX), int(iCY), int(iStride), oOptions.PreviewWidth, oOptions.PreviewHeight)
					fnFrame(oImage)
					dNextPreviewMs = dElapsedMs + dPreviewIntervalMs
				}

				if dSnapshotIntervalMs > 0 && dElapsedMs >= dNextSnapshotMs {
					oImage := lockedBufferToRGBA(pData, int(iCX), int(iCY), int(iStride))
					// Encoding a full-size JPEG is slow; do not hold up the capture loop.
					go saveSnapshot(oOptions.SnapshotDir, oImage)
					dNextSnapshotMs = dElapsedMs + dSnapshotIntervalMs
				}

				bufferUnlock(pBuffer)
			}
			comRelease(pBuffer)
		}

		if pWriter != nil {
			sampleSetSampleTime(pSample, iNextTimestamp)
			sampleSetSampleDuration(pSample, iFrameDuration100ns)
			iNextTimestamp += iFrameDuration100ns
			writerWriteSample(pWriter, iWriterStreamIndex, pSample)
		}

		comRelease(pSample)
	}
}
