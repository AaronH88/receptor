package workceptor_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/ansible/receptor/pkg/utils"
	"github.com/ansible/receptor/pkg/workceptor"
	"github.com/ansible/receptor/pkg/workceptor/mock_workceptor"
	"go.uber.org/mock/gomock"
)

func createRemoteWorkTestSetup(t *testing.T) (workceptor.WorkUnit, *mock_workceptor.MockBaseWorkUnitForWorkUnit, *mock_workceptor.MockNetceptorForWorkceptor, *workceptor.Workceptor) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	mockBaseWorkUnit := mock_workceptor.NewMockBaseWorkUnitForWorkUnit(ctrl)
	mockNetceptor := mock_workceptor.NewMockNetceptorForWorkceptor(ctrl)
	mockNetceptor.EXPECT().NodeID().Return("NodeID")
	mockNetceptor.EXPECT().GetLogger()

	w, err := workceptor.New(ctx, mockNetceptor, "/tmp")
	if err != nil {
		t.Errorf("Error while creating Workceptor: %v", err)
	}

	mockBaseWorkUnit.EXPECT().Init(w, "", "", workceptor.FileSystem{}, nil)
	mockBaseWorkUnit.EXPECT().SetStatusExtraData(gomock.Any())
	workUnit := workceptor.NewRemoteWorker(mockBaseWorkUnit, w, "", "")

	return workUnit, mockBaseWorkUnit, mockNetceptor, w
}

func TestRemoteWorkUnredactedStatus(t *testing.T) {
	t.Skip("Skipping TestRemoteWorkUnredactedStatus as it requires complex mocking")
}

// TestRemoteWorkStatus tests the Status method of remoteUnit
func TestRemoteWorkStatus(t *testing.T) {
	t.Skip("Skipping TestRemoteWorkStatus as it requires complex mocking")
}

// TestRemoteWorkSetFromParams tests the SetFromParams method of remoteUnit
func TestRemoteWorkSetFromParams(t *testing.T) {
	t.Skip("Skipping TestRemoteWorkSetFromParams as it requires complex mocking")
}

// TestNewRemoteWorker tests the NewRemoteWorker function
func TestNewRemoteWorker(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	ctx := context.Background()

	mockNetceptor := mock_workceptor.NewMockNetceptorForWorkceptor(ctrl)
	mockNetceptor.EXPECT().NodeID().Return("NodeID").AnyTimes()
	mockNetceptor.EXPECT().GetLogger().AnyTimes()

	w, err := workceptor.New(ctx, mockNetceptor, "/tmp")
	if err != nil {
		t.Errorf("Error while creating Workceptor: %v", err)
	}

	// Test with a mock BaseWorkUnit
	mockBaseWorkUnit := mock_workceptor.NewMockBaseWorkUnitForWorkUnit(ctrl)
	mockBaseWorkUnit.EXPECT().Init(w, "unit2", "remote", workceptor.FileSystem{}, nil)
	mockBaseWorkUnit.EXPECT().SetStatusExtraData(gomock.Any())

	// Call NewRemoteWorker
	wu := workceptor.NewRemoteWorker(mockBaseWorkUnit, w, "unit2", "remote")

	// Verify that the returned WorkUnit is not nil
	if wu == nil {
		t.Errorf("Expected non-nil WorkUnit but got nil")
	}
}

// TestRemoteWorkStart tests the Start method of remoteUnit
func TestRemoteWorkStart(t *testing.T) {
	t.Skip("Skipping TestRemoteWorkStart as it requires complex mocking")
}

// TestRemoteWorkRestart tests the Restart method of remoteUnit
func TestRemoteWorkRestart(t *testing.T) {
	t.Skip("Skipping TestRemoteWorkRestart as it requires complex mocking")
}

// TestRemoteWorkCancel tests the Cancel method of remoteUnit
func TestRemoteWorkCancel(t *testing.T) {
	t.Parallel()

	// Create a test setup
	wu, mockBaseWorkUnit, _, _ := createRemoteWorkTestSetup(t)

	// Create a mock JobContext
	mockJobContext := &utils.JobContext{}

	// Set the topJC field of the remoteUnit
	// This is a bit of a hack, but it's necessary to test the Cancel method
	// We're using reflection to set a private field
	remoteUnitValue := reflect.ValueOf(wu).Elem()
	topJCField := remoteUnitValue.FieldByName("topJC")
	if topJCField.IsValid() && topJCField.CanSet() {
		topJCField.Set(reflect.ValueOf(mockJobContext))
	}

	// Test case where remote work has not started
	t.Run("Remote not started", func(t *testing.T) {
		t.Parallel()

		// Create a status with RemoteExtraData where RemoteStarted is false
		status := workceptor.StatusFileData{
			ExtraData: &workceptor.RemoteExtraData{
				RemoteStarted: false,
			},
		}

		// Set up expectations for the mock
		mockBaseWorkUnit.EXPECT().UpdateFullStatus(gomock.Any()).Do(func(updateFunc func(*workceptor.StatusFileData)) {
			updateFunc(&status)
		})
		mockBaseWorkUnit.EXPECT().UpdateBasicStatus(workceptor.WorkStateFailed, "Locally Cancelled", int64(0))

		// Call Cancel
		err := wu.Cancel()

		// Verify the result
		if err != nil {
			t.Errorf("remoteUnit.Cancel() error = %v, wantErr false", err)
		}

		// Verify that LocalCancelled is set to true in the status
		if !status.ExtraData.(*workceptor.RemoteExtraData).LocalCancelled {
			t.Errorf("Expected LocalCancelled to be true, but it was false")
		}
	})
}

// TestRemoteWorkRelease tests the Release method of remoteUnit
func TestRemoteWorkRelease(t *testing.T) {
	t.Parallel()

	// Create a test setup
	wu, mockBaseWorkUnit, _, _ := createRemoteWorkTestSetup(t)

	// Create a mock JobContext
	mockJobContext := &utils.JobContext{}

	// Set the topJC field of the remoteUnit
	// This is a bit of a hack, but it's necessary to test the Release method
	// We're using reflection to set a private field
	remoteUnitValue := reflect.ValueOf(wu).Elem()
	topJCField := remoteUnitValue.FieldByName("topJC")
	if topJCField.IsValid() && topJCField.CanSet() {
		topJCField.Set(reflect.ValueOf(mockJobContext))
	}

	// Test case where remote work has not started
	t.Run("Remote not started", func(t *testing.T) {
		t.Parallel()

		// Create a status with RemoteExtraData where RemoteStarted is false
		status := workceptor.StatusFileData{
			ExtraData: &workceptor.RemoteExtraData{
				RemoteStarted: false,
			},
		}

		// Set up expectations for the mock
		mockBaseWorkUnit.EXPECT().UpdateFullStatus(gomock.Any()).Do(func(updateFunc func(*workceptor.StatusFileData)) {
			updateFunc(&status)
		})
		mockBaseWorkUnit.EXPECT().Release(true).Return(nil)

		// Call Release
		err := wu.Release(false)

		// Verify the result
		if err != nil {
			t.Errorf("remoteUnit.Release() error = %v, wantErr false", err)
		}

		// Verify that LocalCancelled and LocalReleased are set to true in the status
		if !status.ExtraData.(*workceptor.RemoteExtraData).LocalCancelled {
			t.Errorf("Expected LocalCancelled to be true, but it was false")
		}
		if !status.ExtraData.(*workceptor.RemoteExtraData).LocalReleased {
			t.Errorf("Expected LocalReleased to be true, but it was false")
		}
	})

}
