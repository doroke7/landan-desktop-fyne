package inputApplicationRecognitionIrInference

import (
	pbRecognitionTableInference "landan-desktop-fyne/pb/recognition/table/inference"
)

type DiskHandler struct {
	pbRecognitionTableInference.UnimplementedDiskServiceServer
}

func NewDiskHandler() *DiskHandler {
	return &DiskHandler{}
}
