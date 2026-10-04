package inputApplicationRecognitionIrInference

import (
	pbRecognitionTableInference "landan-desktop-fyne/pb/recognition/table/inference"
)

type PokerHandler struct {
	pbRecognitionTableInference.UnimplementedPokerServiceServer
}

func NewPokerHandler() *PokerHandler {
	return &PokerHandler{}
}
