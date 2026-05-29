package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/conf"
	"lehu-video/app/videoApi/service/internal/data"
	"lehu-video/app/videoApi/service/internal/server"
	"lehu-video/app/videoApi/service/internal/service"
)

func wireCampusApp(confServer *conf.Server, registry *conf.Registry, confData *conf.Data, auth *conf.Auth, logger log.Logger) (*kratos.App, func(), error) {
	registrar, err := data.NewRegistrar(registry)
	if err != nil {
		return nil, nil, err
	}
	discovery, err := data.NewDiscovery(registry)
	if err != nil {
		return nil, nil, err
	}

	accountServiceClient, err := data.NewAccountServiceClient(discovery)
	if err != nil {
		return nil, nil, err
	}
	authServiceClient, err := data.NewAuthServiceClient(discovery)
	if err != nil {
		return nil, nil, err
	}
	fileServiceClient, err := data.NewFileServiceClient(discovery)
	if err != nil {
		return nil, nil, err
	}
	baseAdapter := data.NewBaseAdapter(accountServiceClient, authServiceClient, fileServiceClient)

	userServiceClient, err := data.NewUserServiceClient(discovery)
	if err != nil {
		return nil, nil, err
	}
	coreAdapter := data.NewCampusCoreAdapter(userServiceClient, logger)

	authSecret := biz.NewAuthSecret(auth)
	userUsecase := biz.NewUserUsecase(baseAdapter, coreAdapter, nil, authSecret, logger)
	userServiceService := service.NewUserServiceService(userUsecase)
	fileUsecase := biz.NewFileUsecase(baseAdapter, logger)
	fileServiceService := service.NewFileServiceService(fileUsecase)

	redisClient, err := data.NewRedis(confData)
	if err != nil {
		return nil, nil, err
	}
	db, err := data.NewDB(confData, logger)
	if err != nil {
		_ = redisClient.Close()
		return nil, nil, err
	}
	dataData, cleanup, err := data.NewData(db, redisClient, logger)
	if err != nil {
		_ = redisClient.Close()
		return nil, nil, err
	}

	campusRepo := data.NewCampusRepo(dataData, logger)
	campusTimetableProvider := biz.NewMockCampusTimetableProvider()
	campusIDGenerator, err := biz.NewCampusIDGenerator()
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	campusRAGClient := biz.NewCampusRAGClient(logger)
	campusUsecase := biz.NewCampusUsecase(campusRepo, baseAdapter, coreAdapter, campusTimetableProvider, campusIDGenerator, campusRAGClient, authSecret, logger)
	campusService := service.NewCampusService(campusUsecase, authSecret, logger)
	httpServer := server.NewCampusHTTPServer(confServer, auth, userServiceService, fileServiceService, campusService, dataData, logger)
	campusTaskServer := server.NewCampusTaskServer(campusUsecase, logger)
	app := newCampusApp(logger, registrar, httpServer, campusTaskServer)
	return app, cleanup, nil
}
