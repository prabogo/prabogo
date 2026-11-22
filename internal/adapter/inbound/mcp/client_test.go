package mcp_inbound_adapter_test

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"

	mcp_inbound_adapter "prabogo/internal/adapter/inbound/mcp"
	"prabogo/internal/domain"
	"prabogo/internal/model"
	mock_outbound_port "prabogo/tests/mocks/port"
)

func TestClientAdapter(t *testing.T) {
	Convey("Test Client MCP Adapter", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockDatabasePort := mock_outbound_port.NewMockDatabasePort(mockCtrl)
		mockMessagePort := mock_outbound_port.NewMockMessagePort(mockCtrl)
		mockCachePort := mock_outbound_port.NewMockCachePort(mockCtrl)
		mockWorkflowPort := mock_outbound_port.NewMockWorkflowPort(mockCtrl)
		mockClientDatabasePort := mock_outbound_port.NewMockClientDatabasePort(mockCtrl)

		mockDatabasePort.EXPECT().Client().Return(mockClientDatabasePort).AnyTimes()
		mockMessagePort.EXPECT().Client().Return(mock_outbound_port.NewMockClientMessagePort(mockCtrl)).AnyTimes()
		mockCachePort.EXPECT().Client().Return(mock_outbound_port.NewMockClientCachePort(mockCtrl)).AnyTimes()
		mockWorkflowPort.EXPECT().Client().Return(mock_outbound_port.NewMockClientWorkflowPort(mockCtrl)).AnyTimes()

		dom := domain.NewDomain(mockDatabasePort, mockMessagePort, mockCachePort, mockWorkflowPort)
		adapter := mcp_inbound_adapter.NewAdapter(dom)

		filter := model.ClientFilter{IDs: []int{1}}
		outputs := []model.Client{
			{
				ID: 1,
				ClientInput: model.ClientInput{
					Name:      "Test Client",
					BearerKey: "test-bearer-key",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
		}

		Convey("Find success with value payload", func() {
			mockClientDatabasePort.EXPECT().FindByFilter(gomock.Any(), gomock.Any(), false).Return(outputs, nil).Times(1)

			result, err := adapter.Client().Find(filter)
			So(err, ShouldBeNil)
			So(result, ShouldResemble, outputs)
		})

		Convey("Find pointer payload should fail", func() {
			result, err := adapter.Client().Find(&filter)
			So(err, ShouldNotBeNil)
			So(result, ShouldBeNil)
		})

		Convey("Find invalid payload", func() {
			result, err := adapter.Client().Find("invalid")
			So(err, ShouldNotBeNil)
			So(result, ShouldBeNil)
		})

		Convey("Find domain error", func() {
			mockClientDatabasePort.EXPECT().FindByFilter(gomock.Any(), gomock.Any(), false).Return(nil, errors.New("database error")).Times(1)

			result, err := adapter.Client().Find(filter)
			So(err, ShouldNotBeNil)
			So(result, ShouldBeNil)
		})
	})
}
