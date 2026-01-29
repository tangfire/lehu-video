package utils

import pb "lehu-video/api/videoChat/service/v1"

func GetSuccessMeta() *pb.Metadata {
	return &pb.Metadata{
		Code:    0,
		Message: "success",
	}
}

func GetMetaWithError(err error) *pb.Metadata {
	return &pb.Metadata{
		Code:    -1,
		Message: err.Error(),
	}
}
