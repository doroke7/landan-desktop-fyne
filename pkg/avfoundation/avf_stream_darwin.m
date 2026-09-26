//go:build darwin && cgo

#import <AVFoundation/AVFoundation.h>
#import <Accelerate/Accelerate.h>
#import <CoreMedia/CoreMedia.h>
#import <CoreVideo/CoreVideo.h>
#include <stdlib.h>
#include <string.h>

#include "avf_stream_darwin.h"

#pragma clang diagnostic ignored "-Wdeprecated-declarations"

// Frames dropped before the first snapshot so the camera's auto exposure can settle (the first frames are dark).
static const int kSnapshotWarmupFrames = 10;

@interface CameraStream : NSObject <AVCaptureVideoDataOutputSampleBufferDelegate>
@property(nonatomic) uintptr_t handle;

@property(nonatomic, strong) AVCaptureSession *session;
@property(nonatomic, strong) dispatch_queue_t queue;
@property(nonatomic, strong) id errorObserver;

// recording (created when the first frame arrives, so it uses the real frame size)
@property(nonatomic, copy) NSString *recordPath;
@property(nonatomic) int bitrate;
@property(nonatomic, strong) AVAssetWriter *writer;
@property(nonatomic, strong) AVAssetWriterInput *writerInput;
@property(nonatomic) BOOL recordingFailed;

// when the camera was switched to an exact format, frames of any other size (from before the switch) are dropped
@property(nonatomic) int frameWidth;
@property(nonatomic) int frameHeight;

// preview
@property(nonatomic) int previewWidth;
@property(nonatomic) int previewHeight;
@property(nonatomic) double previewInterval;
@property(nonatomic) double nextPreview;
@property(nonatomic) unsigned char *previewBuffer;

// YUV -> RGBA of the current frame, shared by the preview and the snapshot
@property(nonatomic) unsigned char *rgbaBuffer;
@property(nonatomic) size_t rgbaSize;

// snapshots
@property(nonatomic) double snapshotInterval; // 0 = off
@property(nonatomic) double nextSnapshot;
@property(nonatomic) int frameCount;

- (void)report:(NSString *)sMessage;
- (void)finish;
@end

@implementation CameraStream

- (void)dealloc {
    free(self.previewBuffer);
    free(self.rgbaBuffer);
}

- (void)report:(NSString *)sMessage {
    goCameraError(self.handle, (char *)[sMessage UTF8String]);
}

- (BOOL)startWriterWithSample:(CMSampleBufferRef)oSample {
    CVImageBufferRef oImage = CMSampleBufferGetImageBuffer(oSample);
    if (oImage == NULL) {
        return NO;
    }

    NSURL *oURL = [NSURL fileURLWithPath:self.recordPath];
    [[NSFileManager defaultManager] removeItemAtURL:oURL error:nil];

    NSError *oError = nil;
    AVAssetWriter *oWriter = [AVAssetWriter assetWriterWithURL:oURL fileType:AVFileTypeMPEG4 error:&oError];
    if (oWriter == nil) {
        [self report:[NSString stringWithFormat:@"無法建立錄影檔: %@", oError.localizedDescription]];
        return NO;
    }
    // A fragmented mp4 stays playable even if the process dies before it is finalized.
    oWriter.movieFragmentInterval = CMTimeMake(2, 1);

    NSDictionary *aSettings = @{
        AVVideoCodecKey : AVVideoCodecTypeH264,
        AVVideoWidthKey : @(CVPixelBufferGetWidth(oImage)),
        AVVideoHeightKey : @(CVPixelBufferGetHeight(oImage)),
        AVVideoCompressionPropertiesKey : @{AVVideoAverageBitRateKey : @(self.bitrate)},
    };
    AVAssetWriterInput *oInput = [AVAssetWriterInput assetWriterInputWithMediaType:AVMediaTypeVideo outputSettings:aSettings];
    oInput.expectsMediaDataInRealTime = YES;
    if (![oWriter canAddInput:oInput]) {
        [self report:@"無法設定錄影格式"];
        return NO;
    }
    [oWriter addInput:oInput];

    if (![oWriter startWriting]) {
        [self report:[NSString stringWithFormat:@"無法開始錄影: %@", oWriter.error.localizedDescription]];
        return NO;
    }
    [oWriter startSessionAtSourceTime:CMSampleBufferGetPresentationTimeStamp(oSample)];

    self.writer = oWriter;
    self.writerInput = oInput;
    return YES;
}

- (void)recordSample:(CMSampleBufferRef)oSample {
    if (self.recordingFailed) {
        return;
    }
    if (self.writer == nil && ![self startWriterWithSample:oSample]) {
        self.recordingFailed = YES;
        return;
    }
    if (self.writer.status == AVAssetWriterStatusFailed) {
        self.recordingFailed = YES;
        [self report:[NSString stringWithFormat:@"錄影失敗: %@", self.writer.error.localizedDescription]];
        return;
    }
    if (self.writerInput.readyForMoreMediaData) {
        [self.writerInput appendSampleBuffer:oSample];
    }
}

// toRGBA converts the locked YUV frame to RGBA (byte order R,G,B,A) in a reused buffer.
- (BOOL)convert:(CVImageBufferRef)oImage into:(vImage_Buffer *)oRGBA {
    static vImage_YpCbCrToARGB sConversion;
    static dispatch_once_t sOnce;
    static BOOL sReady;
    dispatch_once(&sOnce, ^{
        vImage_YpCbCrPixelRange oRange = {16, 128, 235, 240, 255, 0, 255, 0}; // 8-bit video range
        sReady = vImageConvert_YpCbCrToARGB_GenerateConversion(kvImage_YpCbCrToARGBMatrix_ITU_R_709_2, &oRange, &sConversion,
                                                               kvImage420Yp8_CbCr8, kvImageARGB8888, kvImageNoFlags) == kvImageNoError;
    });
    if (!sReady || CVPixelBufferGetPlaneCount(oImage) != 2) {
        return NO;
    }

    size_t iWidth = CVPixelBufferGetWidth(oImage);
    size_t iHeight = CVPixelBufferGetHeight(oImage);
    size_t iSize = iWidth * 4 * iHeight;
    if (self.rgbaSize != iSize) {
        free(self.rgbaBuffer);
        self.rgbaBuffer = malloc(iSize);
        self.rgbaSize = self.rgbaBuffer != NULL ? iSize : 0;
    }
    if (self.rgbaBuffer == NULL) {
        return NO;
    }

    vImage_Buffer oY = {CVPixelBufferGetBaseAddressOfPlane(oImage, 0), CVPixelBufferGetHeightOfPlane(oImage, 0),
                        CVPixelBufferGetWidthOfPlane(oImage, 0), CVPixelBufferGetBytesPerRowOfPlane(oImage, 0)};
    vImage_Buffer oCbCr = {CVPixelBufferGetBaseAddressOfPlane(oImage, 1), CVPixelBufferGetHeightOfPlane(oImage, 1),
                           CVPixelBufferGetWidthOfPlane(oImage, 1), CVPixelBufferGetBytesPerRowOfPlane(oImage, 1)};
    *oRGBA = (vImage_Buffer){self.rgbaBuffer, iHeight, iWidth, iWidth * 4};

    // Output channel i takes ARGB channel map[i]: {1,2,3,0} writes R,G,B,A, the order Go's image.RGBA wants.
    const uint8_t aMap[4] = {1, 2, 3, 0};
    return vImageConvert_420Yp8_CbCr8ToARGB8888(&oY, &oCbCr, oRGBA, &sConversion, aMap, 255, kvImageNoFlags) == kvImageNoError;
}

// sendPreview scales the RGBA frame down and hands it to Go.
- (void)sendPreview:(vImage_Buffer *)oRGBA {
    int iDstStride = self.previewWidth * 4;

    if (self.previewBuffer == NULL) {
        self.previewBuffer = malloc((size_t)iDstStride * self.previewHeight);
        if (self.previewBuffer == NULL) {
            return;
        }
    }

    vImage_Buffer oDst = {self.previewBuffer, (vImagePixelCount)self.previewHeight, (vImagePixelCount)self.previewWidth, (size_t)iDstStride};
    if (vImageScale_ARGB8888(oRGBA, &oDst, NULL, kvImageNoFlags) != kvImageNoError) {
        return;
    }
    goCameraFrame(self.handle, AVF_FRAME_PREVIEW, self.previewBuffer, self.previewWidth, self.previewHeight, iDstStride);
}

- (void)captureOutput:(AVCaptureOutput *)oOutput
    didOutputSampleBuffer:(CMSampleBufferRef)oSample
           fromConnection:(AVCaptureConnection *)oConnection {
    @autoreleasepool {
        CVImageBufferRef oImage = CMSampleBufferGetImageBuffer(oSample);
        if (oImage == NULL) {
            return;
        }

        if (self.frameWidth > 0 && (CVPixelBufferGetWidth(oImage) != (size_t)self.frameWidth ||
                                    CVPixelBufferGetHeight(oImage) != (size_t)self.frameHeight)) {
            return;
        }

        self.frameCount++;
        double dTime = CMTimeGetSeconds(CMSampleBufferGetPresentationTimeStamp(oSample));

        [self recordSample:oSample];

        BOOL bPreview = dTime >= self.nextPreview;
        BOOL bSnapshot = self.snapshotInterval > 0 && self.frameCount > kSnapshotWarmupFrames && dTime >= self.nextSnapshot;
        if (!bPreview && !bSnapshot) {
            return;
        }

        CVPixelBufferLockBaseAddress(oImage, kCVPixelBufferLock_ReadOnly);
        vImage_Buffer oRGBA;
        if ([self convert:oImage into:&oRGBA]) {
            if (bPreview) {
                // Small tolerance so a 15 fps preview of a 30 fps camera keeps every second frame instead of drifting.
                self.nextPreview = dTime + self.previewInterval - 0.005;
                [self sendPreview:&oRGBA];
            }
            if (bSnapshot) {
                self.nextSnapshot = dTime + self.snapshotInterval;
                goCameraFrame(self.handle, AVF_FRAME_SNAPSHOT, oRGBA.data, (int)oRGBA.width, (int)oRGBA.height, (int)oRGBA.rowBytes);
            }
        }
        CVPixelBufferUnlockBaseAddress(oImage, kCVPixelBufferLock_ReadOnly);
    }
}

// finish closes the mp4 so it is complete. Runs after the camera stopped and the queue drained.
- (void)finish {
    if (self.writer == nil || self.writer.status != AVAssetWriterStatusWriting) {
        return;
    }
    [self.writerInput markAsFinished];

    dispatch_semaphore_t oDone = dispatch_semaphore_create(0);
    [self.writer finishWritingWithCompletionHandler:^{
        dispatch_semaphore_signal(oDone);
    }];
    dispatch_semaphore_wait(oDone, dispatch_time(DISPATCH_TIME_NOW, 6 * NSEC_PER_SEC));
}

@end

static char *copy_error(NSString *sMessage) {
    return strdup([sMessage UTF8String]);
}

// find_camera resolves "default", an index, or a name fragment to a video device.
static AVCaptureDevice *find_camera(const char *cDevice) {
    NSString *sDevice = [NSString stringWithUTF8String:cDevice];
    if (sDevice.length == 0 || [sDevice isEqualToString:@"default"]) {
        return [AVCaptureDevice defaultDeviceWithMediaType:AVMediaTypeVideo];
    }

    AVCaptureDeviceDiscoverySession *oSession = [AVCaptureDeviceDiscoverySession
        discoverySessionWithDeviceTypes:@[ AVCaptureDeviceTypeBuiltInWideAngleCamera, AVCaptureDeviceTypeExternalUnknown ]
                              mediaType:AVMediaTypeVideo
                               position:AVCaptureDevicePositionUnspecified];
    NSArray<AVCaptureDevice *> *aDevices = oSession.devices;

    NSScanner *oScanner = [NSScanner scannerWithString:sDevice];
    NSInteger iIndex = 0;
    if ([oScanner scanInteger:&iIndex] && oScanner.atEnd) {
        return (iIndex >= 0 && iIndex < (NSInteger)aDevices.count) ? aDevices[iIndex] : nil;
    }

    for (AVCaptureDevice *oDevice in aDevices) {
        if ([oDevice.localizedName rangeOfString:sDevice options:NSCaseInsensitiveSearch].location != NSNotFound) {
            return oDevice;
        }
    }
    return nil;
}

static NSString *preset_for(int iWidth, int iHeight) {
    if (iWidth == 1920 && iHeight == 1080) {
        return AVCaptureSessionPreset1920x1080;
    }
    if (iWidth == 1280 && iHeight == 720) {
        return AVCaptureSessionPreset1280x720;
    }
    if (iWidth == 640 && iHeight == 480) {
        return AVCaptureSessionPreset640x480;
    }
    return AVCaptureSessionPresetHigh;
}

// ensure_permission asks for camera access when it was never decided. It returns NULL when access is granted.
static NSString *ensure_permission(int iTimeoutMs) {
    AVAuthorizationStatus iStatus = [AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeVideo];

    if (iStatus == AVAuthorizationStatusNotDetermined) {
        dispatch_semaphore_t oAsked = dispatch_semaphore_create(0);
        [AVCaptureDevice requestAccessForMediaType:AVMediaTypeVideo
                                 completionHandler:^(BOOL bGranted) {
                                     dispatch_semaphore_signal(oAsked);
                                 }];
        if (dispatch_semaphore_wait(oAsked, dispatch_time(DISPATCH_TIME_NOW, (int64_t)iTimeoutMs * NSEC_PER_MSEC)) != 0) {
            return @"等不到攝影機權限的回應,請確認是否有跳出權限對話框";
        }
        iStatus = [AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeVideo];
    }

    if (iStatus != AVAuthorizationStatusAuthorized) {
        return @"沒有攝影機權限,請到 系統設定 > 隱私權與安全性 > 攝影機,允許啟動這個程式的終端機";
    }
    return nil;
}

// set_frame_rate asks the camera for iFps if its current format supports it; otherwise the camera keeps its own rate.
static void set_frame_rate(AVCaptureDevice *oCamera, int iFps) {
    for (AVFrameRateRange *oRange in oCamera.activeFormat.videoSupportedFrameRateRanges) {
        if (iFps < oRange.minFrameRate || iFps > oRange.maxFrameRate) {
            continue;
        }
        NSError *oError = nil;
        if ([oCamera lockForConfiguration:&oError]) {
            oCamera.activeVideoMinFrameDuration = CMTimeMake(1, iFps);
            oCamera.activeVideoMaxFrameDuration = CMTimeMake(1, iFps);
            [oCamera unlockForConfiguration];
        }
        return;
    }
}

// choose_format switches the camera to a format that is exactly iWidth x iHeight and supports iFps,
// so frames arrive at that size without scaling. A session preset is not enough: the camera's own
// default format (e.g. 1920x1080) can win over it. It returns NO when the camera has no such format.
static BOOL choose_format(AVCaptureDevice *oCamera, int iWidth, int iHeight, int iFps) {
    for (AVCaptureDeviceFormat *oFormat in oCamera.formats) {
        CMVideoDimensions oSize = CMVideoFormatDescriptionGetDimensions(oFormat.formatDescription);
        if (oSize.width != iWidth || oSize.height != iHeight) {
            continue;
        }
        for (AVFrameRateRange *oRange in oFormat.videoSupportedFrameRateRanges) {
            if (iFps < oRange.minFrameRate || iFps > oRange.maxFrameRate) {
                continue;
            }
            NSError *oError = nil;
            if (![oCamera lockForConfiguration:&oError]) {
                return NO;
            }
            oCamera.activeFormat = oFormat;
            oCamera.activeVideoMinFrameDuration = CMTimeMake(1, iFps);
            oCamera.activeVideoMaxFrameDuration = CMTimeMake(1, iFps);
            [oCamera unlockForConfiguration];
            return YES;
        }
    }
    return NO;
}

void *avf_stream_start(const char *cDevice, int iWidth, int iHeight, int iFps, int iBitrate,
                       int iPreviewWidth, int iPreviewHeight, int iPreviewFps,
                       const char *cRecordPath, double dSnapshotSeconds,
                       uintptr_t iHandle, int iTimeoutMs, char **oError) {
    @autoreleasepool {
        NSString *sProblem = ensure_permission(iTimeoutMs);
        if (sProblem != nil) {
            *oError = copy_error(sProblem);
            return NULL;
        }

        AVCaptureDevice *oCamera = find_camera(cDevice);
        if (oCamera == nil) {
            *oError = copy_error([NSString stringWithFormat:@"找不到攝影機 %s", cDevice]);
            return NULL;
        }

        NSError *oInputError = nil;
        AVCaptureDeviceInput *oInput = [AVCaptureDeviceInput deviceInputWithDevice:oCamera error:&oInputError];
        if (oInput == nil) {
            *oError = copy_error([NSString stringWithFormat:@"無法開啟攝影機 %@: %@", oCamera.localizedName,
                                                            oInputError.localizedDescription]);
            return NULL;
        }

        AVCaptureSession *oSession = [[AVCaptureSession alloc] init];
        if (![oSession canAddInput:oInput]) {
            *oError = copy_error(@"攝影機無法加入擷取流程");
            return NULL;
        }
        [oSession addInput:oInput];

        CameraStream *oStream = [[CameraStream alloc] init];
        oStream.handle = iHandle;
        oStream.session = oSession;
        oStream.queue = dispatch_queue_create("camera-stream", DISPATCH_QUEUE_SERIAL);
        oStream.recordPath = [NSString stringWithUTF8String:cRecordPath];
        oStream.bitrate = iBitrate;
        oStream.previewWidth = iPreviewWidth;
        oStream.previewHeight = iPreviewHeight;
        oStream.previewInterval = 1.0 / iPreviewFps;
        oStream.snapshotInterval = dSnapshotSeconds;

        AVCaptureVideoDataOutput *oOutput = [[AVCaptureVideoDataOutput alloc] init];
        // The camera's own YUV format: the recording goes to the hardware encoder without any conversion.
        oOutput.videoSettings = @{(id)kCVPixelBufferPixelFormatTypeKey : @(kCVPixelFormatType_420YpCbCr8BiPlanarVideoRange)};
        oOutput.alwaysDiscardsLateVideoFrames = YES;
        [oOutput setSampleBufferDelegate:oStream queue:oStream.queue];
        if (![oSession canAddOutput:oOutput]) {
            *oError = copy_error(@"無法取得攝影機影像輸出");
            return NULL;
        }
        [oSession addOutput:oOutput];

        // Set after the input is attached, otherwise the device's own default format can win.
        NSString *sPreset = preset_for(iWidth, iHeight);
        if ([oSession canSetSessionPreset:sPreset]) {
            oSession.sessionPreset = sPreset;
        }

        // The camera can be unplugged or taken over while running.
        oStream.errorObserver = [[NSNotificationCenter defaultCenter]
            addObserverForName:AVCaptureSessionRuntimeErrorNotification
                        object:oSession
                         queue:nil
                    usingBlock:^(NSNotification *oNote) {
                        NSError *oRuntime = oNote.userInfo[AVCaptureSessionErrorKey];
                        [oStream report:[NSString stringWithFormat:@"攝影機中斷: %@", oRuntime.localizedDescription]];
                    }];

        [oSession startRunning];

        // On macOS the session applies its preset when it starts, and that can leave the camera on its own default
        // format (e.g. 1920x1080). Switching the format afterwards sticks, so frames arrive at exactly the wanted size.
        if (choose_format(oCamera, iWidth, iHeight, iFps)) {
            oStream.frameWidth = iWidth;
            oStream.frameHeight = iHeight;
        } else {
            set_frame_rate(oCamera, iFps);
        }
        return (void *)CFBridgingRetain(oStream);
    }
}

void avf_stream_stop(void *pStream) {
    @autoreleasepool {
        CameraStream *oStream = CFBridgingRelease(pStream);

        [[NSNotificationCenter defaultCenter] removeObserver:oStream.errorObserver];
        [oStream.session stopRunning];

        // Let a callback that is still running finish, then nothing else touches the writer.
        dispatch_sync(oStream.queue, ^{
        });
        [oStream finish];
    }
}
