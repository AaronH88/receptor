package workceptor_test

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/ansible/receptor/pkg/logger"
	"github.com/ansible/receptor/pkg/workceptor"
	"github.com/ansible/receptor/pkg/workceptor/mock_workceptor"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/mock/gomock"
)

func TestIsComplete(t *testing.T) {
	testCases := []struct {
		name       string
		workState  int
		isComplete bool
	}{
		{"Pending Work is Incomplete", workceptor.WorkStatePending, false},
		{"Running Work is Incomplete", workceptor.WorkStateRunning, false},
		{"Succeeded Work is Complete", workceptor.WorkStateSucceeded, true},
		{"Failed Work is Complete", workceptor.WorkStateFailed, true},
		{"Unknown Work is Incomplete", 999, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if result := workceptor.IsComplete(tc.workState); result != tc.isComplete {
				t.Errorf("expected %v, got %v", tc.isComplete, result)
			}
		})
	}
}

func TestWorkStateToString(t *testing.T) {
	testCases := []struct {
		name        string
		workState   int
		description string
	}{
		{"Pending Work Description", workceptor.WorkStatePending, "Pending"},
		{"Running Work Description", workceptor.WorkStateRunning, "Running"},
		{"Succeeded Work Description", workceptor.WorkStateSucceeded, "Succeeded"},
		{"Failed Work Description", workceptor.WorkStateFailed, "Failed"},
		{"Canceled Work Description", workceptor.WorkStateCanceled, "Canceled"},
		{"Unknown Work Description", 999, "Unknown: 999"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if result := workceptor.WorkStateToString(tc.workState); result != tc.description {
				t.Errorf("expected %s, got %s", tc.description, result)
			}
		})
	}
}

func TestIsPending(t *testing.T) {
	testCases := []struct {
		name      string
		err       error
		isPending bool
	}{
		{"Pending Error", workceptor.ErrPending, true},
		{"Non-pending Error", errors.New("test error"), false},
		{"Nil Error", nil, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if result := workceptor.IsPending(tc.err); result != tc.isPending {
				t.Errorf("expected %v, got %v", tc.isPending, result)
			}
		})
	}
}

func setUp(t *testing.T) (*gomock.Controller, workceptor.BaseWorkUnit, *workceptor.Workceptor, *mock_workceptor.MockNetceptorForWorkceptor) {
	ctrl := gomock.NewController(t)

	mockNetceptor := mock_workceptor.NewMockNetceptorForWorkceptor(ctrl)

	// attach logger to the mock netceptor and return any number of times
	logger := logger.NewReceptorLogger("")
	mockNetceptor.EXPECT().GetLogger().AnyTimes().Return(logger)
	mockNetceptor.EXPECT().NodeID().Return("NodeID")
	ctx := context.Background()
	w, err := workceptor.New(ctx, mockNetceptor, "/tmp")
	if err != nil {
		t.Errorf("Error while creating Workceptor: %v", err)
	}

	bwu := workceptor.BaseWorkUnit{}

	return ctrl, bwu, w, mockNetceptor
}

func TestInit(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "test", workceptor.FileSystem{}, nil)
	ctrl.Finish()
}

func TestErrorLog(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "test", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	bwu.Error("test error")
	ctrl.Finish()
}

func TestWarningLog(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "test", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	bwu.Warning("test warning")
	ctrl.Finish()
}

func TestInfoLog(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "test", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	bwu.Info("test info")
	ctrl.Finish()
}

func TestDebugLog(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "test", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	bwu.Error("test debug")
	ctrl.Finish()
}

func TestSetFromParams(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "test", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	err := bwu.SetFromParams(nil)
	if err != nil {
		t.Errorf("SetFromParams should return nil: got %v", err)
	}
	ctrl.Finish()
}

const (
	rootDir  = "/tmp"
	testDir  = "NodeID/test"
	dirError = "no such file or directory"
)

func TestUnitDir(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "test", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	expectedUnitDir := path.Join(rootDir, testDir)
	if unitDir := bwu.UnitDir(); unitDir != expectedUnitDir {
		t.Errorf("UnitDir returned wrong value: got %s, want %s", unitDir, expectedUnitDir)
	}
	ctrl.Finish()
}

func TestID(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "test", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	if id := bwu.ID(); id != "test" {
		t.Errorf("ID returned wrong value: got %s, want %s", id, "test")
	}
	ctrl.Finish()
}

func TestStatusFileName(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	expectedUnitDir := path.Join(rootDir, testDir)
	expectedStatusFileName := path.Join(expectedUnitDir, "status")
	if statusFileName := bwu.StatusFileName(); statusFileName != expectedStatusFileName {
		t.Errorf("StatusFileName returned wrong value: got %s, want %s", statusFileName, expectedStatusFileName)
	}
	ctrl.Finish()
}

func TestStdoutFileName(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	expectedUnitDir := path.Join(rootDir, testDir)
	expectedStdoutFileName := path.Join(expectedUnitDir, "stdout")
	if stdoutFileName := bwu.StdoutFileName(); stdoutFileName != expectedStdoutFileName {
		t.Errorf("StdoutFileName returned wrong value: got %s, want %s", stdoutFileName, expectedStdoutFileName)
	}
	ctrl.Finish()
}

func TestBaseSave(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	err := bwu.Save()
	if !strings.Contains(err.Error(), dirError) {
		t.Errorf("Base Work Unit Save, no such file or directory expected, instead %s", err.Error())
	}
	ctrl.Finish()
}

func TestBaseLoad(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	err := bwu.Load()
	if !strings.Contains(err.Error(), dirError) {
		t.Errorf("TestBaseLoad, no such file or directory expected, instead %s", err.Error())
	}
	ctrl.Finish()
}

func TestBaseUpdateFullStatus(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	sf := func(sfd *workceptor.StatusFileData) {
		// Do nothing
	}
	bwu.UpdateFullStatus(sf)
	err := bwu.LastUpdateError()
	if !strings.Contains(err.Error(), dirError) {
		t.Errorf("TestBaseUpdateFullStatus, no such file or directory expected, instead %s", err.Error())
	}
	ctrl.Finish()
}

func TestBaseUpdateBasicStatus(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	bwu.UpdateBasicStatus(1, "Details", 0)
	err := bwu.LastUpdateError()
	if !strings.Contains(err.Error(), dirError) {
		t.Errorf("TestBaseUpdateBasicStatus, no such file or directory expected, instead %s", err.Error())
	}
	ctrl.Finish()
}

func TestBaseStatus(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	status := bwu.Status()
	if status.State != workceptor.WorkStatePending {
		t.Errorf("TestBaseStatus, expected work state pending, received %d", status.State)
	}
	ctrl.Finish()
}

func TestBaseRelease(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	mockFileSystem := mock_workceptor.NewMockFileSystemer(ctrl)
	bwu.Init(w, "test", "", mockFileSystem, &workceptor.RealWatcher{})

	const removeError = "RemoveAll Error"
	testCases := []struct {
		name  string
		err   error
		force bool
		calls func()
	}{
		{
			name:  removeError,
			err:   errors.New(removeError),
			force: false,
			calls: func() { mockFileSystem.EXPECT().RemoveAll(gomock.Any()).Return(errors.New(removeError)).Times(3) },
		},
		{
			name:  "No remote error without force",
			err:   nil,
			force: false,
			calls: func() { mockFileSystem.EXPECT().RemoveAll(gomock.Any()).Return(nil) },
		},
		{
			name:  "No remote error with force",
			err:   nil,
			force: true,
			calls: func() { mockFileSystem.EXPECT().RemoveAll(gomock.Any()).Return(nil) },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.calls()
			err := bwu.Release(tc.force)
			if err != nil && err.Error() != tc.err.Error() {
				t.Errorf("Error returned dosent match, err received %s, expected %s", err, tc.err)
			}
		})
	}

	ctrl.Finish()
}

func TestMonitorLocalStatus(t *testing.T) {
	tests := []struct {
		name          string
		statObj       *Info
		statObjLater  *Info
		addWatcherErr error
		statErr       error
		fsNotifyEvent *fsnotify.Event // using pointer to allow nil
		sleepDuration time.Duration
	}{
		{
			name:          "Handle Write Event",
			statObj:       NewInfo("test", 1, 0, time.Now()),
			addWatcherErr: nil,
			statErr:       nil,
			fsNotifyEvent: &fsnotify.Event{Op: fsnotify.Write},
			sleepDuration: 100 * time.Millisecond,
		},
		{
			name:          "Error Adding Watcher",
			statObj:       NewInfo("test", 1, 0, time.Now()),
			addWatcherErr: fmt.Errorf("error adding watcher"),
			statErr:       nil,
			fsNotifyEvent: nil,
			sleepDuration: 100 * time.Millisecond,
		},
		{
			name:          "Error Reading Status",
			statObj:       nil,
			addWatcherErr: fmt.Errorf("error adding watcher"),
			statErr:       fmt.Errorf("stat error"),
			fsNotifyEvent: nil,
			sleepDuration: 100 * time.Millisecond,
		},
		{
			name:          "Handle Context Cancellation",
			statObj:       NewInfo("test", 1, 0, time.Now()),
			addWatcherErr: nil,
			statErr:       nil,
			fsNotifyEvent: &fsnotify.Event{Op: fsnotify.Write},
			sleepDuration: 100 * time.Millisecond,
		},
		{
			name:          "Handle File Update Without Event",
			statObj:       NewInfo("test", 1, 0, time.Now()),
			statObjLater:  NewInfo("test", 1, 0, time.Now().Add(10*time.Second)),
			addWatcherErr: nil,
			statErr:       nil,
			fsNotifyEvent: &fsnotify.Event{Op: fsnotify.Write},
			sleepDuration: 500 * time.Millisecond,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl, bwu, w, _ := setUp(t)
			defer ctrl.Finish()

			mockWatcher := mock_workceptor.NewMockWatcherWrapper(ctrl)
			mockFileSystem := mock_workceptor.NewMockFileSystemer(ctrl)
			bwu.Init(w, "test", "", mockFileSystem, mockWatcher)

			mockFileSystem.EXPECT().Stat(gomock.Any()).Return(tc.statObj, tc.statErr).AnyTimes()
			if tc.statObjLater != nil {
				mockFileSystem.EXPECT().Stat(gomock.Any()).Return(tc.statObjLater, nil).AnyTimes()
			}
			mockWatcher.EXPECT().Add(gomock.Any()).Return(tc.addWatcherErr)
			mockWatcher.EXPECT().Close().AnyTimes()

			if tc.fsNotifyEvent != nil {
				eventCh := make(chan fsnotify.Event, 1)
				mockWatcher.EXPECT().EventChannel().Return(eventCh).AnyTimes()
				go func() { eventCh <- *tc.fsNotifyEvent }()
			}

			go bwu.MonitorLocalStatus()

			time.Sleep(tc.sleepDuration)
			bwu.CancelContext()
		})
	}
}

// TestIsComplete tests the IsComplete function for various work states.
// func TestIsComplete(t *testing.T) {
// 	testCases := []struct {
// 		name       string
// 		workState  int
// 		isComplete bool
// 	}{
// 		{"Pending Work is Incomplete", workceptor.WorkStatePending, false},
// 		{"Running Work is Incomplete", workceptor.WorkStateRunning, false},
// 		{"Succeeded Work is Complete", workceptor.WorkStateSucceeded, true},
// 		{"Failed Work is Complete", workceptor.WorkStateFailed, true},
// 		{"Canceled Work is Complete", workceptor.WorkStateCanceled, true},
// 		{"Unknown Work is Incomplete", 999, false},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			if result := workceptor.IsComplete(tc.workState); result != tc.isComplete {
// 				t.Errorf("expected %v, got %v", tc.isComplete, result)
// 			}
// 		})
// 	}
// }

// TestWorkStateToString tests the WorkStateToString function for various work states.
// func TestWorkStateToString(t *testing.T) {
// 	testCases := []struct {
// 		name        string
// 		workState   int
// 		description string
// 	}{
// 		{"Pending Work Description", workceptor.WorkStatePending, "Pending"},
// 		{"Running Work Description", workceptor.WorkStateRunning, "Running"},
// 		{"Succeeded Work Description", workceptor.WorkStateSucceeded, "Succeeded"},
// 		{"Failed Work Description", workceptor.WorkStateFailed, "Failed"},
// 		{"Canceled Work Description", workceptor.WorkStateCanceled, "Canceled"},
// 		{"Unknown Work Description", 999, "Unknown: 999"},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			if result := workceptor.WorkStateToString(tc.workState); result != tc.description {
// 				t.Errorf("expected %s, got %s", tc.description, result)
// 			}
// 		})
// 	}
// }

// // TestIsPending tests the IsPending function for various error inputs.
// func TestIsPending(t *testing.T) {
// 	testCases := []struct {
// 		name      string
// 		err       error
// 		isPending bool
// 	}{
// 		{"Pending Error", workceptor.ErrPending, true},
// 		{"Non-pending Error", errors.New("test error"), false},
// 		{"Nil Error", nil, false},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			if result := workceptor.IsPending(tc.err); result != tc.isPending {
// 				t.Errorf("expected %v, got %v", tc.isPending, result)
// 			}
// 		})
// 	}
// }

// // setUp initializes the testing environment.
// func setUp(t *testing.T) (*gomock.Controller, workceptor.BaseWorkUnit, *workceptor.Workceptor, *mock_workceptor.MockNetceptorForWorkceptor) {
// 	ctrl := gomock.NewController(t)

// 	mockNetceptor := mock_workceptor.NewMockNetceptorForWorkceptor(ctrl)

// 	// Attach logger to the mock netceptor and return any number of times
// 	logger := logger.NewReceptorLogger("")
// 	mockNetceptor.EXPECT().GetLogger().AnyTimes().Return(logger)
// 	mockNetceptor.EXPECT().NodeID().Return("NodeID")
// 	ctx := context.Background()
// 	w, err := workceptor.New(ctx, mockNetceptor, "/tmp")
// 	if err != nil {
// 		t.Fatalf("Error while creating Workceptor: %v", err)
// 	}

// 	bwu := workceptor.BaseWorkUnit{}

// 	return ctrl, bwu, w, mockNetceptor
// }

// // TestInit tests the Init function of BaseWorkUnit.
// func TestInit(t *testing.T) {
// 	ctrl, bwu, w, _ := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "testType", workceptor.FileSystem{}, nil)

// 	if bwu.ID() != "test" {
// 		t.Errorf("Expected unit ID 'test', got '%s'", bwu.ID())
// 	}
// 	if bwu.Status().WorkType != "testType" {
// 		t.Errorf("Expected work type 'testType', got '%s'", bwu.Status().WorkType)
// 	}
// }

// // TestErrorLog tests the Error logging function.
// func TestErrorLog(t *testing.T) {
// 	ctrl, bwu, w, mockNetceptor := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "test", workceptor.FileSystem{}, &workceptor.RealWatcher{})

// 	mockNetceptor.EXPECT().GetLogger().Return(logger.NewReceptorLogger("")).AnyTimes()
// 	bwu.Error("test error")
// }

// // TestWarningLog tests the Warning logging function.
// func TestWarningLog(t *testing.T) {
// 	ctrl, bwu, w, mockNetceptor := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "test", workceptor.FileSystem{}, &workceptor.RealWatcher{})

// 	mockNetceptor.EXPECT().GetLogger().Return(logger.NewReceptorLogger("")).AnyTimes()
// 	bwu.Warning("test warning")
// }

// // TestInfoLog tests the Info logging function.
// func TestInfoLog(t *testing.T) {
// 	ctrl, bwu, w, mockNetceptor := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "test", workceptor.FileSystem{}, &workceptor.RealWatcher{})

// 	mockNetceptor.EXPECT().GetLogger().Return(logger.NewReceptorLogger("")).AnyTimes()
// 	bwu.Info("test info")
// }

// // TestDebugLog tests the Debug logging function.
// func TestDebugLog(t *testing.T) {
// 	ctrl, bwu, w, mockNetceptor := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "test", workceptor.FileSystem{}, &workceptor.RealWatcher{})

// 	mockNetceptor.EXPECT().GetLogger().Return(logger.NewReceptorLogger("")).AnyTimes()
// 	bwu.Debug("test debug")
// }

// // TestSetFromParams tests the SetFromParams function.
// func TestSetFromParams(t *testing.T) {
// 	ctrl, bwu, w, _ := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "test", workceptor.FileSystem{}, &workceptor.RealWatcher{})
// 	err := bwu.SetFromParams(nil)
// 	if err != nil {
// 		t.Errorf("SetFromParams should return nil: got %v", err)
// 	}
// }

// const (
// 	rootDir  = "/tmp"
// 	testDir  = "test"
// 	dirError = "no such file or directory"
// )

// // TestUnitDir tests the UnitDir function.
// func TestUnitDir(t *testing.T) {
// 	ctrl, bwu, w, _ := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "test", workceptor.FileSystem{}, &workceptor.RealWatcher{})
// 	expectedUnitDir := path.Join(rootDir, testDir)
// 	if unitDir := bwu.UnitDir(); unitDir != expectedUnitDir {
// 		t.Errorf("UnitDir returned wrong value: got %s, want %s", unitDir, expectedUnitDir)
// 	}
// }

// // TestID tests the ID function.
// func TestID(t *testing.T) {
// 	ctrl, bwu, w, _ := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "test", workceptor.FileSystem{}, &workceptor.RealWatcher{})
// 	if id := bwu.ID(); id != "test" {
// 		t.Errorf("ID returned wrong value: got %s, want %s", id, "test")
// 	}
// }

// // TestStatusFileName tests the StatusFileName function.
// func TestStatusFileName(t *testing.T) {
// 	ctrl, bwu, w, _ := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
// 	expectedUnitDir := path.Join(rootDir, testDir)
// 	expectedStatusFileName := path.Join(expectedUnitDir, "status")
// 	if statusFileName := bwu.StatusFileName(); statusFileName != expectedStatusFileName {
// 		t.Errorf("StatusFileName returned wrong value: got %s, want %s", statusFileName, expectedStatusFileName)
// 	}
// }

// // TestStdoutFileName tests the StdoutFileName function.
// func TestStdoutFileName(t *testing.T) {
// 	ctrl, bwu, w, _ := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
// 	expectedUnitDir := path.Join(rootDir, testDir)
// 	expectedStdoutFileName := path.Join(expectedUnitDir, "stdout")
// 	if stdoutFileName := bwu.StdoutFileName(); stdoutFileName != expectedStdoutFileName {
// 		t.Errorf("StdoutFileName returned wrong value: got %s, want %s", stdoutFileName, expectedStdoutFileName)
// 	}
// }

// // TestBaseSave tests the Save function when the directory does not exist.
// func TestBaseSave(t *testing.T) {
// 	ctrl, bwu, w, _ := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
// 	err := bwu.Save()
// 	if err == nil || !strings.Contains(err.Error(), dirError) {
// 		t.Errorf("Expected error containing '%s', got '%v'", dirError, err)
// 	}
// }

// // TestBaseLoad tests the Load function when the directory does not exist.
// func TestBaseLoad(t *testing.T) {
// 	ctrl, bwu, w, _ := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
// 	err := bwu.Load()
// 	if err == nil || !strings.Contains(err.Error(), dirError) {
// 		t.Errorf("Expected error containing '%s', got '%v'", dirError, err)
// 	}
// }

// // TestBaseUpdateFullStatus tests the UpdateFullStatus function when the directory does not exist.
// func TestBaseUpdateFullStatus(t *testing.T) {
// 	ctrl, bwu, w, _ := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
// 	sf := func(sfd *workceptor.StatusFileData) {
// 		// Do nothing
// 	}
// 	bwu.UpdateFullStatus(sf)
// 	err := bwu.LastUpdateError()
// 	if err == nil || !strings.Contains(err.Error(), dirError) {
// 		t.Errorf("Expected error containing '%s', got '%v'", dirError, err)
// 	}
// }

// // TestBaseUpdateBasicStatus tests the UpdateBasicStatus function when the directory does not exist.
// func TestBaseUpdateBasicStatus(t *testing.T) {
// 	ctrl, bwu, w, _ := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
// 	bwu.UpdateBasicStatus(1, "Details", 0)
// 	err := bwu.LastUpdateError()
// 	if err == nil || !strings.Contains(err.Error(), dirError) {
// 		t.Errorf("Expected error containing '%s', got '%v'", dirError, err)
// 	}
// }

// // TestBaseStatus tests the Status function.
// func TestBaseStatus(t *testing.T) {
// 	ctrl, bwu, w, _ := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
// 	status := bwu.Status()
// 	if status.State != workceptor.WorkStatePending {
// 		t.Errorf("Expected work state %d, received %d", workceptor.WorkStatePending, status.State)
// 	}
// }

// // TestUnredactedStatus tests the UnredactedStatus function.
// func TestUnredactedStatus(t *testing.T) {
// 	ctrl, bwu, w, _ := setUp(t)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
// 	extraData := "Sensitive Data"
// 	bwu.SetStatusExtraData(extraData)
// 	status := bwu.UnredactedStatus()
// 	if status.ExtraData != extraData {
// 		t.Errorf("Expected ExtraData '%v', got '%v'", extraData, status.ExtraData)
// 	}
// }

// TestSetStatusExtraData tests the SetStatusExtraData function.
func TestSetStatusExtraData(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	defer ctrl.Finish()
	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	extraData := "Sensitive Data"
	bwu.SetStatusExtraData(extraData)
	if bwu.UnredactedStatus().ExtraData != extraData {
		t.Errorf("Expected ExtraData '%v', got '%v'", extraData, bwu.UnredactedStatus().ExtraData)
	}
}

// TestGetStatusCopy tests the GetStatusCopy function.
func TestGetStatusCopy(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	defer ctrl.Finish()
	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	statusCopy := bwu.GetStatusCopy()
	if statusCopy.State != workceptor.WorkStatePending {
		t.Errorf("Expected State %d, got %d", workceptor.WorkStatePending, statusCopy.State)
	}
}

// TestGetStatusWithoutExtraData tests the GetStatusWithoutExtraData function.
func TestGetStatusWithoutExtraData(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	defer ctrl.Finish()
	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	extraData := "Sensitive Data"
	bwu.SetStatusExtraData(extraData)
	status := bwu.GetStatusWithoutExtraData()
	if status.ExtraData != nil {
		t.Errorf("Expected ExtraData nil, got '%v'", status.ExtraData)
	}
}

// TestGetStatusLock tests the GetStatusLock function.
func TestGetStatusLock(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	defer ctrl.Finish()
	bwu.Init(w, "test", "", workceptor.FileSystem{}, &workceptor.RealWatcher{})
	lock := bwu.GetStatusLock()
	if lock == nil {
		t.Errorf("Expected a non-nil status lock")
	}
}

// TestGetSetWorkceptor tests the GetWorkceptor and SetWorkceptor functions.
func TestGetSetWorkceptor(t *testing.T) {
	ctrl, bwu, _, _ := setUp(t)
	defer ctrl.Finish()
	w := &workceptor.Workceptor{}
	bwu.SetWorkceptor(w)
	if bwu.GetWorkceptor() != w {
		t.Errorf("Expected workceptor to be set correctly")
	}
}

// TestGetContextGetCancel tests the GetContext and GetCancel functions.
func TestGetContextGetCancel(t *testing.T) {
	ctrl, bwu, _, _ := setUp(t)
	defer ctrl.Finish()
	ctx := bwu.GetContext()
	cancel := bwu.GetCancel()
	if ctx == nil || cancel == nil {
		t.Errorf("Expected non-nil context and cancel function")
	}
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(done)
	}()
	cancel()
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Errorf("Context was not canceled within 1 second")
	}
}

// TestBaseRelease tests the Release function.
// func TestBaseRelease(t *testing.T) {
// 	ctrl, bwu, w, _ := setUp(t)
// 	mockFileSystem := mock_workceptor.NewMockFileSystemer(ctrl)
// 	defer ctrl.Finish()
// 	bwu.Init(w, "test", "", mockFileSystem, &workceptor.RealWatcher{})

// 	const removeError = "RemoveAll Error"
// 	testCases := []struct {
// 		name   string
// 		err    error
// 		force  bool
// 		calls  func()
// 		expect error
// 	}{
// 		{
// 			name:   removeError,
// 			err:    errors.New(removeError),
// 			force:  false,
// 			calls:  func() { mockFileSystem.EXPECT().RemoveAll(gomock.Any()).Return(errors.New(removeError)).Times(3) },
// 			expect: errors.New(removeError),
// 		},
// 		{
// 			name:   "No remove error without force",
// 			err:    nil,
// 			force:  false,
// 			calls:  func() { mockFileSystem.EXPECT().RemoveAll(gomock.Any()).Return(nil) },
// 			expect: nil,
// 		},
// 		{
// 			name:   "No remove error with force",
// 			err:    nil,
// 			force:  true,
// 			calls:  func() { mockFileSystem.EXPECT().RemoveAll(gomock.Any()).Return(nil) },
// 			expect: nil,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			tc.calls()
// 			err := bwu.Release(tc.force)
// 			if err != nil && tc.expect != nil && err.Error() != tc.expect.Error() {
// 				t.Errorf("Expected error '%v', got '%v'", tc.expect, err)
// 			}
// 		})
// 	}
// }

// TestCancelContext tests the CancelContext function.
func TestCancelContext(t *testing.T) {
	ctrl, bwu, _, _ := setUp(t)
	defer ctrl.Finish()
	done := make(chan struct{})
	go func() {
		<-bwu.GetContext().Done()
		close(done)
	}()
	bwu.CancelContext()
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Errorf("Context was not canceled within 1 second")
	}
}

// TestMonitorLocalStatus tests the MonitorLocalStatus function.
// func TestMonitorLocalStatus(t *testing.T) {
// 	tests := []struct {
// 		name          string
// 		addWatcherErr error
// 		statErr       error
// 		fsNotifyEvent *fsnotify.Event
// 	}{
// 		{
// 			name:          "Handle Write Event",
// 			addWatcherErr: nil,
// 			statErr:       nil,
// 			fsNotifyEvent: &fsnotify.Event{Op: fsnotify.Write},
// 		},
// 		{
// 			name:          "Error Adding Watcher",
// 			addWatcherErr: fmt.Errorf("error adding watcher"),
// 			statErr:       nil,
// 			fsNotifyEvent: nil,
// 		},
// 		{
// 			name:          "Error Reading Status",
// 			addWatcherErr: nil,
// 			statErr:       fmt.Errorf("stat error"),
// 			fsNotifyEvent: nil,
// 		},
// 		{
// 			name:          "Handle Context Cancellation",
// 			addWatcherErr: nil,
// 			statErr:       nil,
// 			fsNotifyEvent: &fsnotify.Event{Op: fsnotify.Write},
// 		},
// 	}

// 	for _, tc := range tests {
// 		t.Run(tc.name, func(t *testing.T) {
// 			ctrl, bwu, w, _ := setUp(t)
// 			defer ctrl.Finish()

// 			mockWatcher := mock_workceptor.NewMockWatcherWrapper(ctrl)
// 			mockFileSystem := mock_workceptor.NewMockFileSystemer(ctrl)
// 			bwu.Init(w, "test", "", mockFileSystem, mockWatcher)

// 			mockFileSystem.EXPECT().Stat(gomock.Any()).Return(nil, tc.statErr).AnyTimes()
// 			mockWatcher.EXPECT().Add(gomock.Any()).Return(tc.addWatcherErr)
// 			mockWatcher.EXPECT().Close().AnyTimes()

// 			if tc.fsNotifyEvent != nil {
// 				eventCh := make(chan fsnotify.Event, 1)
// 				mockWatcher.EXPECT().EventChannel().Return(eventCh).AnyTimes()
// 				go func() { eventCh <- *tc.fsNotifyEvent }()
// 			}

// 			go bwu.MonitorLocalStatus()
// 			time.Sleep(100 * time.Millisecond)
// 			bwu.CancelContext()
// 		})
// 	}
// }

// TestUnknownUnitMethods tests the methods of unknownUnit.
// func TestUnknownUnitMethods(t *testing.T) {
// 	ctrl, _, w, _ := setUp(t)
// 	defer ctrl.Finish()
// 	uu := &workceptor.UnknownUnit{}
// 	uu.Init(w, "unknownID", "unknownType", workceptor.FileSystem{}, nil)

// 	// Test Start
// 	err := uu.Start()
// 	if err != nil {
// 		t.Errorf("Expected Start to return nil, got %v", err)
// 	}

// 	// Test Restart
// 	err = uu.Restart()
// 	if err != nil {
// 		t.Errorf("Expected Restart to return nil, got %v", err)
// 	}

// 	// Test Cancel
// 	err = uu.Cancel()
// 	if err != nil {
// 		t.Errorf("Expected Cancel to return nil, got %v", err)
// 	}

// 	// Test Status
// 	status := uu.Status()
// 	if status.ExtraData != "Unknown WorkType" {
// 		t.Errorf("Expected ExtraData 'Unknown WorkType', got '%v'", status.ExtraData)
// 	}
// }

// TestUpdateFullStatusSuccess tests UpdateFullStatus with a successful update.
func TestUpdateFullStatusSuccess(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	defer ctrl.Finish()
	mockFileSystem := mock_workceptor.NewMockFileSystemer(ctrl)
	bwu.Init(w, "test", "", mockFileSystem, &workceptor.RealWatcher{})

	// Mock the file system interactions
	mockFileSystem.EXPECT().Stat(gomock.Any()).Return(nil, nil).AnyTimes()
	mockFileSystem.EXPECT().RemoveAll(gomock.Any()).Return(nil).AnyTimes()

	bwu.UpdateFullStatus(func(sfd *workceptor.StatusFileData) {
		sfd.State = workceptor.WorkStateSucceeded
	})

	if bwu.Status().State != workceptor.WorkStateSucceeded {
		t.Errorf("Expected State %d, got %d", workceptor.WorkStateSucceeded, bwu.Status().State)
	}
}

// TestUpdateBasicStatusSuccess tests UpdateBasicStatus with a successful update.
func TestUpdateBasicStatusSuccess(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	defer ctrl.Finish()
	mockFileSystem := mock_workceptor.NewMockFileSystemer(ctrl)
	bwu.Init(w, "test", "", mockFileSystem, &workceptor.RealWatcher{})

	// Mock the file system interactions
	mockFileSystem.EXPECT().Stat(gomock.Any()).Return(nil, nil).AnyTimes()
	mockFileSystem.EXPECT().RemoveAll(gomock.Any()).Return(nil).AnyTimes()

	bwu.UpdateBasicStatus(workceptor.WorkStateRunning, "Running", -1)

	if bwu.Status().State != workceptor.WorkStateRunning {
		t.Errorf("Expected State %d, got %d", workceptor.WorkStateRunning, bwu.Status().State)
	}
	if bwu.Status().Detail != "Running" {
		t.Errorf("Expected Detail 'Running', got '%s'", bwu.Status().Detail)
	}
}

// TestLastUpdateError tests the LastUpdateError function.
func TestLastUpdateError(t *testing.T) {
	ctrl, bwu, w, _ := setUp(t)
	defer ctrl.Finish()
	mockFileSystem := mock_workceptor.NewMockFileSystemer(ctrl)
	bwu.Init(w, "test", "", mockFileSystem, &workceptor.RealWatcher{})

	// Mock an error during UpdateBasicStatus
	mockFileSystem.EXPECT().Stat(gomock.Any()).Return(nil, errors.New("stat error")).AnyTimes()
	bwu.UpdateBasicStatus(workceptor.WorkStateRunning, "Running", -1)

	err := bwu.LastUpdateError()
	if err == nil || !strings.Contains(err.Error(), "stat error") {
		t.Errorf("Expected error containing 'stat error', got '%v'", err)
	}
}
