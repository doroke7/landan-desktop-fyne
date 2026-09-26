package inputApplicationIrInference

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	outputPortAnyModel "landan-desktop-fyne/internal/output/port/any/model"
	pbIrTableInference "landan-desktop-fyne/pb/ir/table/inference"
)

type DieHandler struct {
	pbIrTableInference.UnimplementedDieServiceServer
	dieTopDetectorModel     outputPortAnyModel.DieTopDetectorModel
	dieValueClassifierModel outputPortAnyModel.DieValueClassifierModel
}

func NewDieHandler(
	oDieTopDetectorModel outputPortAnyModel.DieTopDetectorModel,
	oDieValueClassifierModel outputPortAnyModel.DieValueClassifierModel,
) *DieHandler {
	return &DieHandler{
		dieTopDetectorModel:     oDieTopDetectorModel,
		dieValueClassifierModel: oDieValueClassifierModel,
	}
}

func (oSelf *DieHandler) Recognize(_ context.Context, oRequest *pbIrTableInference.DieRecognizeRequest) (*pbIrTableInference.DieRecognizeResponse, error) {
	if len(oRequest.GetImage()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "image is empty")
	}

	aDies, err := oSelf.dieTopDetectorModel.Recognize(oRequest.GetImage())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "recognize dice: %v", err)
	}

	aItems := make([]*pbIrTableInference.Die, 0, len(aDies))
	for _, oDie := range aDies {
		aItems = append(aItems, &pbIrTableInference.Die{
			X:          int32(oDie.X),
			Y:          int32(oDie.Y),
			Width:      int32(oDie.Width),
			Height:     int32(oDie.Height),
			Confidence: oDie.Confidence,
		})
	}

	return &pbIrTableInference.DieRecognizeResponse{Items: aItems}, nil
}
