package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yourusername/telegram-bot-microservice/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func TestGRPCServer_MessageProcessing(t *testing.T) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := proto.NewBotServiceClient(conn)

	t.Run("ValidMessage", func(t *testing.T) {
		req := &proto.ProcessMessageRequest{
			Message: "Hello, Bot!",
			UserId:  "12345",
		}
		resp, err := client.ProcessMessage(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, "Response from bot", resp.Response)
	})

	t.Run("InvalidMessage", func(t *testing.T) {
		req := &proto.ProcessMessageRequest{
			Message: "",
			UserId:  "12345",
		}
		resp, err := client.ProcessMessage(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, int32(400), st.Code())
	})

	t.Run("UserNotFound", func(t *testing.T) {
		req := &proto.ProcessMessageRequest{
			Message: "Hello, Bot!",
			UserId:  "unknown_user",
		}
		resp, err := client.ProcessMessage(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, int32(404), st.Code())
	})
}
