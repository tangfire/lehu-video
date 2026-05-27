package utils

import (
	pb "lehu-video/api/videoCore/service/v1"
	"lehu-video/pkg/apperror"
)

func GetSuccessMeta() *pb.Metadata {
	return &pb.Metadata{
		Code:    0,
		Message: "success",
	}
}

func GetMetaWithError(err error) *pb.Metadata {
	appErr := apperror.From(err)
	return &pb.Metadata{
		Code:    appErr.Code,
		Message: appErr.Message,
		Reason:  []string{appErr.Reason},
	}
}
