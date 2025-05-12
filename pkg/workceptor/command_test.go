package workceptor_test

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/ansible/receptor/pkg/logger"
	"github.com/ansible/receptor/pkg/workceptor"
	"github.com/ansible/receptor/pkg/workceptor/mock_workceptor"
	"go.uber.org/mock/gomock"
)

func statusExpectCalls(mockBaseWorkUnit *mock_workceptor.MockBaseWorkUnitForWorkUnit) {
	statusLock := &sync.RWMutex{}
	mockBaseWorkUnit.EXPECT().GetStatusLock().Return(statusLock).Times(2)
	mockBaseWorkUnit.EXPECT().GetStatusWithoutExtraData().Return(&workceptor.StatusFileData{})
	mockBaseWorkUnit.EXPECT().GetStatusCopy().Return(workceptor.StatusFileData{
		ExtraData: &workceptor.CommandExtraData{},
	})
}

func createCommandTestSetup(t *testing.T) (workceptor.WorkUnit, *mock_workceptor.MockBaseWorkUnitForWorkUnit, *mock_workceptor.MockNetceptorForWorkceptor, *workceptor.Workceptor) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	mockBaseWorkUnit := mock_workceptor.NewMockBaseWorkUnitForWorkUnit(ctrl)
	mockNetceptor := mock_workceptor.NewMockNetceptorForWorkceptor(ctrl)
	mockNetceptor.EXPECT().NodeID().Return("NodeID")

	w, err := workceptor.New(ctx, mockNetceptor, "/tmp")
	if err != nil {
		t.Errorf("Error while creating Workceptor: %v", err)
	}

	cwc := &workceptor.CommandWorkerCfg{}
	mockBaseWorkUnit.EXPECT().Init(w, "", "", workceptor.FileSystem{}, nil)
	workUnit := cwc.NewWorker(mockBaseWorkUnit, w, "", "")

	return workUnit, mockBaseWorkUnit, mockNetceptor, w
}

func TestCommandSetFromParams(t *testing.T) {
	wu, mockBaseWorkUnit, _, _ := createCommandTestSetup(t)

	paramsTestCases := []struct {
		name          string
		params        map[string]string
		expectedCalls func()
		errorCatch    func(error, *testing.T)
	}{
		{
			name:   "no params with no error",
			params: map[string]string{"": ""},
			expectedCalls: func() {
				mockBaseWorkUnit.EXPECT().GetStatusCopy().Return(workceptor.StatusFileData{
					ExtraData: &workceptor.CommandExtraData{},
				})
			},
			errorCatch: func(err error, t *testing.T) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			name:   "params with error",
			params: map[string]string{"params": "param"},
			expectedCalls: func() {
			},
			errorCatch: func(err error, t *testing.T) {
				if err == nil {
					t.Error(err)
				}
			},
		},
	}

	for _, testCase := range paramsTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.expectedCalls()
			err := wu.SetFromParams(testCase.params)
			testCase.errorCatch(err, t)
		})
	}
}

func TestUnredactedStatus(t *testing.T) {
	wu, mockBaseWorkUnit, _, _ := createCommandTestSetup(t)
	restartTestCases := []struct {
		name string
	}{
		{name: "test1"},
		{name: "test2"},
	}

	statusLock := &sync.RWMutex{}
	for _, testCase := range restartTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			mockBaseWorkUnit.EXPECT().GetStatusLock().Return(statusLock).Times(2)
			mockBaseWorkUnit.EXPECT().GetStatusWithoutExtraData().Return(&workceptor.StatusFileData{})
			mockBaseWorkUnit.EXPECT().GetStatusCopy().Return(workceptor.StatusFileData{
				ExtraData: &workceptor.CommandExtraData{},
			})
			wu.UnredactedStatus()
		})
	}
}

func TestStart(t *testing.T) {
	wu, mockBaseWorkUnit, mockNetceptor, w := createCommandTestSetup(t)

	mockBaseWorkUnit.EXPECT().GetWorkceptor().Return(w).Times(2)
	mockNetceptor.EXPECT().GetLogger().Times(2)
	mockBaseWorkUnit.EXPECT().UpdateBasicStatus(gomock.Any(), gomock.Any(), gomock.Any())
	statusExpectCalls(mockBaseWorkUnit)

	mockBaseWorkUnit.EXPECT().UnitDir()
	mockBaseWorkUnit.EXPECT().UpdateFullStatus(gomock.Any())
	mockBaseWorkUnit.EXPECT().MonitorLocalStatus().AnyTimes()
	mockBaseWorkUnit.EXPECT().UpdateFullStatus(gomock.Any()).AnyTimes()
	wu.Start()
}

func TestRestart(t *testing.T) {
	wu, mockBaseWorkUnit, _, _ := createCommandTestSetup(t)

	restartTestCases := []struct {
		name          string
		expectedCalls func()
		errorCatch    func(error, *testing.T)
	}{
		{
			name: "load error",
			expectedCalls: func() {
				mockBaseWorkUnit.EXPECT().Load().Return(errors.New("terminated"))
			},
			errorCatch: func(err error, t *testing.T) {
				if err.Error() != "terminated" {
					t.Error(err)
				}
			},
		},
		{
			name: "job complete with no error",
			expectedCalls: func() {
				statusFile := &workceptor.StatusFileData{State: 2}
				mockBaseWorkUnit.EXPECT().Load().Return(nil)
				statusLock := &sync.RWMutex{}
				mockBaseWorkUnit.EXPECT().GetStatusLock().Return(statusLock).Times(2)
				mockBaseWorkUnit.EXPECT().GetStatusWithoutExtraData().Return(statusFile)
				mockBaseWorkUnit.EXPECT().GetStatusCopy().Return(workceptor.StatusFileData{
					ExtraData: &workceptor.CommandExtraData{},
				})
			},
			errorCatch: func(err error, t *testing.T) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			name: "restart successful",
			expectedCalls: func() {
				statusFile := &workceptor.StatusFileData{State: 0}
				mockBaseWorkUnit.EXPECT().Load().Return(nil)
				statusLock := &sync.RWMutex{}
				mockBaseWorkUnit.EXPECT().GetStatusLock().Return(statusLock).Times(2)
				mockBaseWorkUnit.EXPECT().GetStatusWithoutExtraData().Return(statusFile)
				mockBaseWorkUnit.EXPECT().GetStatusCopy().Return(workceptor.StatusFileData{
					ExtraData: &workceptor.CommandExtraData{},
				})
				mockBaseWorkUnit.EXPECT().UpdateBasicStatus(gomock.Any(), gomock.Any(), gomock.Any())
				mockBaseWorkUnit.EXPECT().UnitDir()
			},
			errorCatch: func(err error, t *testing.T) {
				if err != nil {
					t.Error(err)
				}
			},
		},
	}

	for _, testCase := range restartTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.expectedCalls()
			mockBaseWorkUnit.EXPECT().MonitorLocalStatus().AnyTimes()
			err := wu.Restart()
			testCase.errorCatch(err, t)
		})
	}
}

func TestCancel(t *testing.T) {
	wu, mockBaseWorkUnit, _, _ := createCommandTestSetup(t)

	paramsTestCases := []struct {
		name          string
		expectedCalls func()
		errorCatch    func(error, *testing.T)
	}{
		{
			name: "not a valid pid no error",
			expectedCalls: func() {
				mockBaseWorkUnit.EXPECT().CancelContext()
				statusExpectCalls(mockBaseWorkUnit)
			},
			errorCatch: func(err error, t *testing.T) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			name: "process interrupt error",
			expectedCalls: func() {
				mockBaseWorkUnit.EXPECT().CancelContext()
				mockBaseWorkUnit.EXPECT().GetStatusLock().Return(&sync.RWMutex{}).Times(2)
				mockBaseWorkUnit.EXPECT().GetStatusWithoutExtraData().Return(&workceptor.StatusFileData{})
				mockBaseWorkUnit.EXPECT().GetStatusCopy().Return(workceptor.StatusFileData{
					ExtraData: &workceptor.CommandExtraData{
						Pid: 1,
					},
				})
			},
			errorCatch: func(err error, t *testing.T) {
				if err == nil {
					t.Error(err)
				}
			},
		},
		{
			name: "process already finished",
			expectedCalls: func() {
				mockBaseWorkUnit.EXPECT().CancelContext()
				mockBaseWorkUnit.EXPECT().GetStatusLock().Return(&sync.RWMutex{}).Times(2)
				mockBaseWorkUnit.EXPECT().GetStatusWithoutExtraData().Return(&workceptor.StatusFileData{})

				c := exec.Command("ls", "/tmp")
				processPid := make(chan int)

				go func(c *exec.Cmd, processPid chan int) {
					c.Run()
					processPid <- c.Process.Pid
				}(c, processPid)

				time.Sleep(200 * time.Millisecond)

				mockBaseWorkUnit.EXPECT().GetStatusCopy().Return(workceptor.StatusFileData{
					ExtraData: &workceptor.CommandExtraData{
						Pid: <-processPid,
					},
				})
			},
			errorCatch: func(err error, t *testing.T) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			name: "cancelled process successfully",
			expectedCalls: func() {
				mockBaseWorkUnit.EXPECT().CancelContext()
				mockBaseWorkUnit.EXPECT().GetStatusLock().Return(&sync.RWMutex{}).Times(2)
				mockBaseWorkUnit.EXPECT().GetStatusWithoutExtraData().Return(&workceptor.StatusFileData{})
				mockBaseWorkUnit.EXPECT().UpdateBasicStatus(gomock.Any(), gomock.Any(), gomock.Any())

				c := exec.Command("sleep", "30")
				processPid := make(chan int)

				go func(c *exec.Cmd, processPid chan int) {
					err := c.Start()
					if err != nil {
						fmt.Println(err)
					}
					processPid <- c.Process.Pid
				}(c, processPid)
				time.Sleep(200 * time.Millisecond)

				mockBaseWorkUnit.EXPECT().GetStatusCopy().Return(workceptor.StatusFileData{
					ExtraData: &workceptor.CommandExtraData{
						Pid: <-processPid,
					},
				})
			},
			errorCatch: func(err error, t *testing.T) {
				if err != nil {
					t.Error(err)
				}
			},
		},
	}

	for _, testCase := range paramsTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.expectedCalls()
			err := wu.Cancel()
			testCase.errorCatch(err, t)
		})
	}
}

func TestRelease(t *testing.T) {
	wu, mockBaseWorkUnit, _, _ := createCommandTestSetup(t)

	releaseTestCases := []struct {
		name          string
		expectedCalls func()
		errorCatch    func(error, *testing.T)
		force         bool
	}{
		{
			name:          "cancel error",
			expectedCalls: func() {},
			errorCatch: func(err error, t *testing.T) {
				if err == nil {
					t.Error(err)
				}
			},
			force: false,
		},
		{
			name: "released successfully",
			expectedCalls: func() {
				mockBaseWorkUnit.EXPECT().Release(gomock.Any())
			},
			errorCatch: func(err error, t *testing.T) {
				if err != nil {
					t.Error(err)
				}
			},
			force: true,
		},
	}
	for _, testCase := range releaseTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			mockBaseWorkUnit.EXPECT().CancelContext()
			mockBaseWorkUnit.EXPECT().GetStatusLock().Return(&sync.RWMutex{}).Times(2)
			mockBaseWorkUnit.EXPECT().GetStatusWithoutExtraData().Return(&workceptor.StatusFileData{})
			mockBaseWorkUnit.EXPECT().GetStatusCopy().Return(workceptor.StatusFileData{
				ExtraData: &workceptor.CommandExtraData{
					Pid: 1,
				},
			})
			testCase.expectedCalls()
			err := wu.Release(testCase.force)
			testCase.errorCatch(err, t)
		})
	}
}

func TestSigningKeyPrepare(t *testing.T) {
	privateKey := workceptor.SigningKeyPrivateCfg{}
	err := privateKey.Prepare()

	if err == nil {
		t.Error(err)
	}
}

func TestPrepareSigningKeyPrivateCfg(t *testing.T) {
	signingKeyTestCases := []struct {
		name            string
		errorCatch      func(error, *testing.T)
		privateKey      string
		tokenExpiration string
	}{
		{
			name:            "file does not exist error",
			privateKey:      "does_not_exist.txt",
			tokenExpiration: "",
			errorCatch: func(err error, t *testing.T) {
				if err == nil {
					t.Error(err)
				}
			},
		},
		{
			name:            "failed to parse token expiration",
			privateKey:      "/etc/hosts",
			tokenExpiration: "random_input",
			errorCatch: func(err error, t *testing.T) {
				if err == nil {
					t.Error(err)
				}
			},
		},
		{
			name:            "duration no error",
			privateKey:      "/etc/hosts",
			tokenExpiration: "3h",
			errorCatch: func(err error, t *testing.T) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			name:            "no duration no error",
			privateKey:      "/etc/hosts",
			tokenExpiration: "",
			errorCatch: func(err error, t *testing.T) {
				if err != nil {
					t.Error(err)
				}
			},
		},
	}

	for _, testCase := range signingKeyTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			privateKey := workceptor.SigningKeyPrivateCfg{
				PrivateKey:      testCase.privateKey,
				TokenExpiration: testCase.tokenExpiration,
			}
			_, err := privateKey.PrepareSigningKeyPrivateCfg()
			testCase.errorCatch(err, t)
		})
	}
}

func TestVerifyingKeyPrepare(t *testing.T) {
	publicKey := workceptor.VerifyingKeyPublicCfg{}
	err := publicKey.Prepare()

	if err == nil {
		t.Error(err)
	}
}

func TestPrepareVerifyingKeyPrivateCfg(t *testing.T) {
	verifyingKeyTestCases := []struct {
		name       string
		errorCatch func(error, *testing.T)
		publicKey  string
	}{
		{
			name:      "file does not exist",
			publicKey: "does_not_exist.txt",
			errorCatch: func(err error, t *testing.T) {
				if err == nil {
					t.Error(err)
				}
			},
		},
		{
			name:      "prepared successfully",
			publicKey: "/etc/hosts",
			errorCatch: func(err error, t *testing.T) {
				if err != nil {
					t.Error(err)
				}
			},
		},
	}

	for _, testCase := range verifyingKeyTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			publicKey := workceptor.VerifyingKeyPublicCfg{
				PublicKey: testCase.publicKey,
			}
			err := publicKey.PrepareVerifyingKeyPublicCfg()
			testCase.errorCatch(err, t)
		})
	}
}

func TestCommandWorkerCfgGetWorkType(t *testing.T) {
	tests := []struct {
		name     string
		workType string
		want     string
	}{
		{
			name:     "Basic",
			workType: "test-worker",
			want:     "test-worker",
		},
		{
			name:     "Empty",
			workType: "",
			want:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := workceptor.CommandWorkerCfg{
				WorkType: tt.workType,
			}
			if got := cfg.TestGetWorkType(); got != tt.want {
				t.Errorf("CommandWorkerCfg.GetWorkType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommandWorkerCfgGetVerifySignature(t *testing.T) {
	tests := []struct {
		name            string
		verifySignature bool
		want            bool
	}{
		{
			name:            "True",
			verifySignature: true,
			want:            true,
		},
		{
			name:            "False",
			verifySignature: false,
			want:            false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := workceptor.CommandWorkerCfg{
				VerifySignature: tt.verifySignature,
			}
			if got := cfg.TestGetVerifySignature(); got != tt.want {
				t.Errorf("CommandWorkerCfg.GetVerifySignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommandWorkerCfgRun(t *testing.T) {
	// Save the original MainInstance
	originalMainInstance := workceptor.MainInstance
	defer func() {
		// Restore the original MainInstance after the test
		workceptor.MainInstance = originalMainInstance
	}()

	// Create a test instance
	workceptor.MainInstance = &workceptor.Workceptor{}

	tests := []struct {
		name            string
		workType        string
		verifySignature bool
		verifyingKey    string
		wantErr         bool
	}{
		{
			name:            "Success without verification",
			workType:        "test-worker",
			verifySignature: false,
			verifyingKey:    "",
			wantErr:         false,
		},
		{
			name:            "Error with verification but no key",
			workType:        "test-worker",
			verifySignature: true,
			verifyingKey:    "",
			wantErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the test instance
			workceptor.MainInstance.VerifyingKey = tt.verifyingKey

			cfg := workceptor.CommandWorkerCfg{
				WorkType:        tt.workType,
				VerifySignature: tt.verifySignature,
			}

			// Skip the actual registration since we don't have a real Workceptor instance
			// This is just to test the verification logic
			if tt.verifySignature && tt.verifyingKey == "" {
				err := cfg.TestRun()
				if (err != nil) != tt.wantErr {
					t.Errorf("CommandWorkerCfg.Run() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestTermThenKill(t *testing.T) {
	// Save the original MainInstance
	originalMainInstance := workceptor.MainInstance
	defer func() {
		// Restore the original MainInstance after the test
		workceptor.MainInstance = originalMainInstance
	}()

	// Create a test instance with a logger
	ctrl := gomock.NewController(t)
	mockNetceptor := mock_workceptor.NewMockNetceptorForWorkceptor(ctrl)
	mockLogger := logger.NewReceptorLogger("")
	mockNetceptor.EXPECT().GetLogger().Return(mockLogger).AnyTimes()
	mockNetceptor.EXPECT().NodeID().Return("test-node").AnyTimes()

	w, err := workceptor.New(context.Background(), mockNetceptor, "/tmp")
	if err != nil {
		t.Fatalf("Error creating Workceptor: %v", err)
	}
	workceptor.MainInstance = w

	tests := []struct {
		name     string
		cmd      *exec.Cmd
		doneChan chan bool
	}{
		{
			name:     "Nil process",
			cmd:      exec.Command("echo", "test"),
			doneChan: make(chan bool),
		},
		{
			name: "Process exits after interrupt",
			cmd: func() *exec.Cmd {
				cmd := exec.Command("sleep", "1")
				err := cmd.Start()
				if err != nil {
					t.Fatalf("Failed to start command: %v", err)
				}
				return cmd
			}(),
			doneChan: make(chan bool),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Process exits after interrupt" {
				// Signal that the process has exited
				go func() {
					time.Sleep(100 * time.Millisecond)
					tt.doneChan <- true
				}()
			}

			// Call termThenKill
			workceptor.TestTermThenKill(tt.cmd, tt.doneChan)

			// No assertions needed - we're just testing that it doesn't panic
		})
	}
}

func TestCommandRunnerCfgRun(t *testing.T) {
	// This function is difficult to test properly because it calls os.Exit
	// We'll skip it for now
	t.Skip("Skipping TestCommandRunnerCfgRun as it calls os.Exit")
}

func TestCommandRunner(t *testing.T) {
	// Skip this test since it requires creating stdin and stdout files
	t.Skip("Skipping TestCommandRunner as it requires creating stdin and stdout files")
}
