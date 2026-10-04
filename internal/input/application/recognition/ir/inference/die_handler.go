package inputApplicationRecognitionIrInference

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usecasePortAnyIrInference "landan-desktop-fyne/internal/usecase/port/any/ir/inference"
	pbRecognitionTableInference "landan-desktop-fyne/pb/recognition/table/inference"
)

type DieHandler struct {
	pbRecognitionTableInference.UnimplementedDieServiceServer
	irInferenceDieUsecase usecasePortAnyIrInference.DieUsecase
}

func NewDieHandler(oDieUsecase usecasePortAnyIrInference.DieUsecase) *DieHandler {
	return &DieHandler{
		irInferenceDieUsecase: oDieUsecase,
	}
}

func (oSelf *DieHandler) Recognize(_ context.Context, oRequest *pbRecognitionTableInference.DieRecognizeRequest) (*pbRecognitionTableInference.DieRecognizeResponse, error) {
	if len(oRequest.GetImage()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "image is empty")
	}

	aDies, err := oSelf.irInferenceDieUsecase.Recognize(oRequest.GetImage())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	aItems := make([]*pbRecognitionTableInference.Die, 0, len(aDies))
	for _, oDie := range aDies {
		aItems = append(aItems, &pbRecognitionTableInference.Die{
			X:          int32(oDie.X),
			Y:          int32(oDie.Y),
			Width:      int32(oDie.Width),
			Height:     int32(oDie.Height),
			Confidence: oDie.Confidence,
		})
	}

	return &pbRecognitionTableInference.DieRecognizeResponse{Items: aItems}, nil
}
