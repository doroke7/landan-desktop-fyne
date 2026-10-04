package registerRecognition

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	container "landan-desktop-fyne/container"

	pbRecognitionTableInference "landan-desktop-fyne/pb/recognition/table/inference"
)

func Init(oContainer *container.RecognitionContainer) *grpc.Server {

	oGrpcServer := grpc.NewServer(
		grpc.KeepaliveParams(
			keepalive.ServerParameters{
				Time:    1 * time.Second,
				Timeout: 5 * time.Second,
			},
		),
		grpc.KeepaliveEnforcementPolicy(
			keepalive.EnforcementPolicy{
				MinTime:             10 * time.Second,
				PermitWithoutStream: true,
			},
		),
	)
	pbRecognitionTableInference.RegisterDieServiceServer(oGrpcServer, oContainer.RecognitionInferenceDie)
	pbRecognitionTableInference.RegisterPokerServiceServer(oGrpcServer, oContainer.RecognitionInferencePoker)
	pbRecognitionTableInference.RegisterDiskServiceServer(oGrpcServer, oContainer.RecognitionInferenceDisk)

	return oGrpcServer
}
