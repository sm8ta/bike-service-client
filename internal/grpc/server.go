package grpc

import (
	"context"
	"webike_bike_microservice_nikita/internal/core/ports"
	"webike_bike_microservice_nikita/internal/core/services"

	webikev1 "github.com/sm8ta/grpc_webike/gen/go/webike"
	"google.golang.org/grpc"
)

type serverAPI struct {
	webikev1.UnimplementedBikeServiceServer
	bikeService *services.BikeService
	log         ports.LoggerPort
}

func Register(
	gRPCServer *grpc.Server,
	bikeService *services.BikeService,
	log ports.LoggerPort,
) {
	webikev1.RegisterBikeServiceServer(gRPCServer, &serverAPI{
		bikeService: bikeService,
		log:         log,
	})
}

func (s *serverAPI) GetBikes(ctx context.Context, req *webikev1.GetBikesRequest) (*webikev1.GetBikesResponse, error) {
	userID := req.GetUserId()

	bikes, err := s.bikeService.GetBikesByUserID(ctx, userID)
	if err != nil {
		s.log.WarnGRPC(ctx, "Failed to get bikes", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		return &webikev1.GetBikesResponse{Bikes: []*webikev1.Bike{}}, nil
	}

	// Convert in to map
	pbBikes := make([]*webikev1.Bike, len(bikes))
	for i, bike := range bikes {
		pbBikes[i] = &webikev1.Bike{
			BikeId:  bike.BikeID.String(),
			Model:   bike.Model,
			Mileage: int32(bike.Mileage),
		}
	}

	s.log.InfoGRPC(ctx, "Bikes retrieved successfully", map[string]interface{}{
		"user_id":     userID,
		"bikes_count": len(bikes),
	})

	return &webikev1.GetBikesResponse{
		Bikes: pbBikes,
	}, nil
}
