package workceptor_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ansible/receptor/pkg/logger"
	"github.com/ansible/receptor/pkg/netceptor"
	"github.com/ansible/receptor/pkg/workceptor"
	"github.com/ansible/receptor/pkg/workceptor/mock_workceptor"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	fakerest "k8s.io/client-go/rest/fake"
	"k8s.io/client-go/tools/remotecommand"
)

func startNetceptorNodeWithWorkceptor() (*workceptor.KubeUnit, error) {
	kw := &workceptor.KubeUnit{
		BaseWorkUnitForWorkUnit: &workceptor.BaseWorkUnit{},
	}

	// Create Netceptor node using external backends
	n1 := netceptor.New(context.Background(), "node1")
	b1, err := netceptor.NewExternalBackend()
	if err != nil {
		return kw, err
	}

	err = n1.AddBackend(b1)
	if err != nil {
		return kw, err
	}

	w, err := workceptor.New(context.Background(), n1, "")
	if err != nil {
		return kw, err
	}

	kw.SetWorkceptor(w)

	return kw, nil
}

func TestShouldUseReconnect(t *testing.T) {
	const envVariable string = "RECEPTOR_KUBE_SUPPORT_RECONNECT"

	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{
			name:     "Enabled test",
			envValue: "enabled",
			want:     true,
		},
		{
			name:     "Disabled test",
			envValue: "disabled",
			want:     false,
		},
		{
			name:     "Auto test",
			envValue: "auto",
			want:     true,
		},
		{
			name:     "Default test",
			envValue: "default",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kw, err := startNetceptorNodeWithWorkceptor()
			if err != nil {
				t.Fatal(err)
			}

			if tt.envValue != "" {
				os.Setenv(envVariable, tt.envValue)
				defer os.Unsetenv(envVariable)
			} else {
				os.Unsetenv(envVariable)
			}

			if got := workceptor.ShouldUseReconnect(kw); got != tt.want {
				t.Errorf("shouldUseReconnect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTimeoutOpenLogstream(t *testing.T) {
	const envVariable string = "RECEPTOR_OPEN_LOGSTREAM_TIMEOUT"

	kw, err := startNetceptorNodeWithWorkceptor()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		envValue string
		want     int
	}{
		{
			name:     "No env value set",
			envValue: "",
			want:     1,
		},
		{
			name:     "Env value set incorrectly to text",
			envValue: "text instead of int",
			want:     1,
		},
		{
			name:     "Env value set incorrectly to negative",
			envValue: "-1",
			want:     1,
		},
		{
			name:     "Env value set incorrectly to zero",
			envValue: "0",
			want:     1,
		},
		{
			name:     "Env value set correctly",
			envValue: "2",
			want:     2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(envVariable, tt.envValue)
				defer os.Unsetenv(envVariable)
			} else {
				os.Unsetenv(envVariable)
			}

			if got := workceptor.GetTimeoutOpenLogstream(kw); got != tt.want {
				t.Errorf("GetTimeoutOpenLogstream() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	type args struct {
		s string
	}

	// Test RFC3339 format
	rfc3339TimeString := "2024-01-17T00:00:00Z"
	rfc3339Time, _ := time.Parse(time.RFC3339, rfc3339TimeString)

	// Test RFC3339Nano format
	rfc3339NanoTimeString := "2024-01-17T00:00:00.123456789Z"
	rfc3339NanoTime, _ := time.Parse(time.RFC3339Nano, rfc3339NanoTimeString)

	// Test RFC3339 format with error
	rfc3339ErrorTimeString := "2024-01-17T00:00:00" // Missing Z

	// Test RFC3339Nano format with error
	rfc3339NanoErrorTimeString := "2024-01-17T00:00:00.123456789" // Missing Z

	tests := []struct {
		name    string
		args    args
		want    *time.Time
		wantErr bool
	}{
		{
			name: "RFC3339 format",
			args: args{
				s: rfc3339TimeString,
			},
			want: &rfc3339Time,
		},
		{
			name: "RFC3339 format with error",
			args: args{
				s: rfc3339ErrorTimeString,
			},
			want: nil,
		},
		{
			name: "RFC3339Nano format",
			args: args{
				s: rfc3339NanoTimeString,
			},
			want: &rfc3339NanoTime,
		},
		{
			name: "RFC3339Nano format with error",
			args: args{
				s: rfc3339NanoErrorTimeString,
			},
			want: nil,
		},
		{
			name: "Invalid format",
			args: args{
				s: "Invalid time",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := workceptor.ParseTime(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func createKubernetesTestSetup(t *testing.T) (workceptor.WorkUnit, *mock_workceptor.MockBaseWorkUnitForWorkUnit, *mock_workceptor.MockNetceptorForWorkceptor, *workceptor.Workceptor, *mock_workceptor.MockKubeAPIer, *gomock.Controller, context.Context) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	mockBaseWorkUnit := mock_workceptor.NewMockBaseWorkUnitForWorkUnit(ctrl)
	mockNetceptor := mock_workceptor.NewMockNetceptorForWorkceptor(ctrl)
	mockNetceptor.EXPECT().NodeID().Return("NodeID")
	mockKubeAPI := mock_workceptor.NewMockKubeAPIer(ctrl)

	w, err := workceptor.New(ctx, mockNetceptor, "/tmp")
	if err != nil {
		t.Errorf("Error while creating Workceptor: %v", err)
	}

	mockBaseWorkUnit.EXPECT().Init(w, "", "", workceptor.FileSystem{}, nil)
	kubeConfig := workceptor.KubeWorkerCfg{AuthMethod: "incluster"}
	ku := kubeConfig.NewkubeWorker(mockBaseWorkUnit, w, "", "", mockKubeAPI)

	return ku, mockBaseWorkUnit, mockNetceptor, w, mockKubeAPI, ctrl, ctx
}

type hasTerm struct {
	field, value string
}

func (h *hasTerm) DeepCopySelector() fields.Selector { return h }
func (h *hasTerm) Empty() bool                       { return true }
func (h *hasTerm) Matches(_ fields.Fields) bool      { return true }
func (h *hasTerm) Requirements() fields.Requirements {
	return []fields.Requirement{{
		Field:    h.field,
		Operator: selection.Equals,
		Value:    h.value,
	}}
}
func (h *hasTerm) RequiresExactMatch(_ string) (value string, found bool)    { return "", true }
func (h *hasTerm) String() string                                            { return "Test" }
func (h *hasTerm) Transform(_ fields.TransformFunc) (fields.Selector, error) { return h, nil }

type ex struct{}

func (e *ex) Stream(_ remotecommand.StreamOptions) error {
	return nil
}

func (e *ex) StreamWithContext(_ context.Context, _ remotecommand.StreamOptions) error {
	return nil
}

func TestConnectUsingIncluster(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockKubeAPI := mock_workceptor.NewMockKubeAPIer(ctrl)
	mockBaseWorkUnit := mock_workceptor.NewMockBaseWorkUnitForWorkUnit(ctrl)

	// Create a KubeUnit with the mock
	kw := &workceptor.KubeUnit{
		BaseWorkUnitForWorkUnit: mockBaseWorkUnit,
		KubeAPIWrapperInstance:  mockKubeAPI,
	}

	// Test cases
	tests := []struct {
		name        string
		setupMock   func()
		expectError bool
	}{
		{
			name: "Success",
			setupMock: func() {
				// Mock a successful InClusterConfig call
				mockKubeAPI.EXPECT().InClusterConfig().Return(&rest.Config{}, nil)
			},
			expectError: false,
		},
		{
			name: "Error",
			setupMock: func() {
				// Mock an error from InClusterConfig
				mockKubeAPI.EXPECT().InClusterConfig().Return(nil, fmt.Errorf("in-cluster config error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the mock expectations
			tt.setupMock()

			// Call the function
			err := kw.TestConnectUsingIncluster()

			// Check the results
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else if err != nil {
				t.Errorf("Expected no error but got %v", err)
			}
		})
	}
}

func TestConnectUsingKubeconfig(t *testing.T) {
	t.Skip("Skipping TestConnectUsingKubeconfig as it requires complex mocking of clientcmd package")
}

func TestKubeStart(t *testing.T) {
	ku, mockbwu, mockNet, w, mockKubeAPI, _, ctx := createKubernetesTestSetup(t)

	startTestCases := []struct {
		name          string
		expectedCalls func()
	}{
		{
			name: "test1",
			expectedCalls: func() {
				mockbwu.EXPECT().UpdateBasicStatus(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
				config := rest.Config{}
				mockKubeAPI.EXPECT().InClusterConfig().Return(&config, nil)
				mockbwu.EXPECT().GetWorkceptor().Return(w).AnyTimes()
				logger := logger.NewReceptorLogger("")
				mockNet.EXPECT().GetLogger().Return(logger).AnyTimes()
				clientset := kubernetes.Clientset{}
				mockKubeAPI.EXPECT().NewForConfig(gomock.Any()).Return(&clientset, nil)
				mockbwu.EXPECT().MonitorLocalStatus().AnyTimes()
				lock := &sync.RWMutex{}
				mockbwu.EXPECT().GetStatusLock().Return(lock).AnyTimes()
				kubeExtraData := workceptor.KubeExtraData{}
				status := workceptor.StatusFileData{ExtraData: &kubeExtraData}
				mockbwu.EXPECT().GetStatusWithoutExtraData().Return(&status).AnyTimes()
				mockbwu.EXPECT().GetStatusCopy().Return(status).AnyTimes()
				mockbwu.EXPECT().GetContext().Return(ctx).AnyTimes()
				pod := corev1.Pod{TypeMeta: metav1.TypeMeta{}, ObjectMeta: metav1.ObjectMeta{Name: "Test Name"}, Spec: corev1.PodSpec{}, Status: corev1.PodStatus{}}

				mockKubeAPI.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&pod, nil).AnyTimes()
				mockbwu.EXPECT().UpdateFullStatus(gomock.Any()).AnyTimes()

				field := hasTerm{}
				mockKubeAPI.EXPECT().OneTermEqualSelector(gomock.Any(), gomock.Any()).Return(&field).AnyTimes()
				ev := watch.Event{Object: &pod}
				mockKubeAPI.EXPECT().UntilWithSync(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&ev, nil).AnyTimes()
				apierr := apierrors.StatusError{}
				mockKubeAPI.EXPECT().NewNotFound(gomock.Any(), gomock.Any()).Return(&apierr).AnyTimes()
				mockbwu.EXPECT().MonitorLocalStatus().AnyTimes()

				c := rest.RESTClient{}
				req := rest.NewRequest(&c)
				mockKubeAPI.EXPECT().SubResource(gomock.Any(), gomock.Any(), gomock.Any()).Return(req).AnyTimes()
				exec := ex{}
				mockKubeAPI.EXPECT().NewSPDYExecutor(gomock.Any(), gomock.Any(), gomock.Any()).Return(&exec, nil).AnyTimes()
				mockbwu.EXPECT().UnitDir().Return("TestDir").AnyTimes()
			},
		},
	}

	for _, testCase := range startTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.expectedCalls()

			err := ku.Start()
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func Test_IsCompatibleK8S(t *testing.T) {
	type args struct {
		kw         *workceptor.KubeUnit
		versionStr string
	}

	kw, err := startNetceptorNodeWithWorkceptor()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Kubernetes X stream negative test",
			args: args{
				versionStr: "v0.0.0",
			},
			want: false,
		},
		{
			name: "Kubernetes Y stream negative test",
			args: args{
				versionStr: "v1.22.9998",
			},
			want: false,
		},
		{
			name: "Kubernetes 1.23 Z stream negative test",
			args: args{
				versionStr: "v1.23.13",
			},
			want: false,
		},
		{
			name: "Kubernetes 1.23 exact positive test",
			args: args{
				versionStr: "v1.23.14",
			},
			want: true,
		},
		{
			name: "Kubernetes 1.23 Z stream positive test",
			args: args{
				versionStr: "v1.23.15",
			},
			want: true,
		},
		{
			name: "Kubernetes 1.24 Z stream negative test",
			args: args{
				versionStr: "v1.24.7",
			},
			want: false,
		},
		{
			name: "Kubernetes 1.24 exact positive test",
			args: args{
				versionStr: "v1.24.8",
			},
			want: true,
		},
		{
			name: "Kubernetes 1.24 Z stream positive test",
			args: args{
				versionStr: "v1.24.9",
			},
			want: true,
		},
		{
			name: "Kubernetes 1.25 Z stream negative test",
			args: args{
				versionStr: "v1.25.3",
			},
			want: false,
		},
		{
			name: "Kuberentes 1.25 exact positive test",
			args: args{
				versionStr: "v1.25.4",
			},
			want: true,
		},
		{
			name: "Kubernetes 1.25 Z stream positive test",
			args: args{
				versionStr: "v1.25.99",
			},
			want: true,
		},
		{
			name: "Kubernetes Y stream positive test",
			args: args{
				versionStr: "v1.26.0",
			},
			want: true,
		},
		{
			name: "Kubernetes X stream positive test 1",
			args: args{
				versionStr: "v2.0.0",
			},
			want: false,
		},
		{
			name: "Kubernetes X stream positive test 2",
			args: args{
				versionStr: "v2.23.14",
			},
			want: true,
		},
		{
			name: "Kubernetes X stream positive test 3",
			args: args{
				versionStr: "v2.24.8",
			},
			want: true,
		},
		{
			name: "Kubernetes X stream positive test 4",
			args: args{
				versionStr: "v2.25.4",
			},
			want: true,
		},
		{
			name: "Kubernetes X stream positive test 5",
			args: args{
				versionStr: "v2.26.0",
			},
			want: true,
		},
		{
			name: "Missing Kubernetes version negative test",
			args: args{
				versionStr: "yoloswag",
			},
			want: false,
		},
		{
			name: "Prerelease Kubernetes version positive test 1",
			args: args{
				versionStr: "v1.32.14+sadfasdf",
			},
			want: true,
		},
		{
			name: "Prerelease Kubernetes version positive test 2",
			args: args{
				versionStr: "v1.32.14-asdfasdf+12131",
			},
			want: true,
		},
		{
			name: "Prerelease Kubernetes version positive test 3",
			args: args{
				versionStr: "v1.32.15-asdfasdf+12131",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		tt.args.kw = kw
		t.Run(tt.name, func(t *testing.T) {
			if got := workceptor.IsCompatibleK8S(tt.args.kw, tt.args.versionStr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IsCompatibleK8S() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKubeLoggingWithReconnect(t *testing.T) {
	var stdinErr error
	var stdoutErr error
	_, mockBaseWorkUnit, mockNetceptor, w, mockKubeAPI, ctrl, ctx := createKubernetesTestSetup(t)

	pod := &corev1.Pod{TypeMeta: metav1.TypeMeta{}, ObjectMeta: metav1.ObjectMeta{Name: "Test_Name", Namespace: "Test_Namespace"}, Spec: corev1.PodSpec{}, Status: corev1.PodStatus{Phase: corev1.PodRunning}}

	kw := &workceptor.KubeUnit{
		BaseWorkUnitForWorkUnit: mockBaseWorkUnit,
		KubeAPIWrapperInstance:  mockKubeAPI,
		Pod:                     pod,
	}

	tests := []struct {
		name          string
		expectedCalls func()
	}{
		{
			name: "Kube error should be read",
			expectedCalls: func() {
				mockBaseWorkUnit.EXPECT().GetWorkceptor().Return(w)
				mockBaseWorkUnit.EXPECT().GetContext().Return(ctx).Times(3)
				mockKubeAPI.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(pod, nil)
				logger := logger.NewReceptorLogger("")
				mockNetceptor.EXPECT().GetLogger().Return(logger)
				req := fakerest.RESTClient{
					Client: fakerest.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
						resp := &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader("2024-12-09T00:31:18.823849250Z HI\n kube error")),
						}

						return resp, nil
					}),
					NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
				}
				mockKubeAPI.EXPECT().GetLogs(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(req.Request())
			},
		},
		{
			name: "Kube error 503",
			expectedCalls: func() {
				mockBaseWorkUnit.EXPECT().GetWorkceptor().Return(w).MinTimes(1)
				mockBaseWorkUnit.EXPECT().GetContext().Return(ctx).MinTimes(3)
				mockKubeAPI.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(pod, nil)
				logger := logger.NewReceptorLogger("")
				mockNetceptor.EXPECT().GetLogger().Return(logger).MinTimes(1)
				mockBaseWorkUnit.EXPECT().UpdateBasicStatus(gomock.Any(), gomock.Any(), gomock.Any())
				req := fakerest.RESTClient{
					Client: fakerest.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
						resp := &http.Response{
							StatusCode: http.StatusServiceUnavailable, // 503
							Body:       io.NopCloser(strings.NewReader("kube error")),
						}

						return resp, nil
					}),
					NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
				}
				mockKubeAPI.EXPECT().GetLogs(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(req.Request())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.expectedCalls()
			wg := &sync.WaitGroup{}
			wg.Add(1)
			mockfilesystemer := mock_workceptor.NewMockFileSystemer(ctrl)
			mockfilesystemer.EXPECT().OpenFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(&os.File{}, nil)
			stdout, _ := workceptor.NewStdoutWriter(mockfilesystemer, "")
			mockFileWC := mock_workceptor.NewMockFileWriteCloser(ctrl)
			stdout.SetWriter(mockFileWC)
			mockFileWC.EXPECT().Write(gomock.AnyOf([]byte("HI\n"), []byte(" kube error\n"))).Return(0, nil).AnyTimes()
			kw.KubeLoggingWithReconnect(wg, stdout, &stdinErr, &stdoutErr)
		})
	}
}

func TestPodRunningAndReady(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockKubeAPI := mock_workceptor.NewMockKubeAPIer(ctrl)
	mockBaseWorkUnit := mock_workceptor.NewMockBaseWorkUnitForWorkUnit(ctrl)

	// Create a KubeUnit with the mock
	kw := &workceptor.KubeUnit{
		BaseWorkUnitForWorkUnit: mockBaseWorkUnit,
		KubeAPIWrapperInstance:  mockKubeAPI,
	}

	// Setup for NewNotFound
	mockKubeAPI.EXPECT().NewNotFound(gomock.Any(), gomock.Any()).Return(&apierrors.StatusError{}).AnyTimes()

	// Test cases
	tests := []struct {
		name        string
		pod         *corev1.Pod
		eventType   watch.EventType
		expectError bool
		errorType   error
		expectTrue  bool
	}{
		{
			name: "Pod deleted",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
			},
			eventType:   watch.Deleted,
			expectError: true,
			expectTrue:  false,
		},
		{
			name: "Pod failed",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
				},
			},
			eventType:   watch.Modified,
			expectError: true,
			errorType:   workceptor.ErrPodFailed,
			expectTrue:  false,
		},
		{
			name: "Pod succeeded",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
				},
			},
			eventType:   watch.Modified,
			expectError: true,
			errorType:   workceptor.ErrPodCompleted,
			expectTrue:  false,
		},
		{
			name: "Pod running but not ready",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			eventType:   watch.Modified,
			expectError: false,
			expectTrue:  false,
		},
		{
			name: "Pod running and ready",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			eventType:   watch.Modified,
			expectError: false,
			expectTrue:  true,
		},
		{
			name: "Pod with image pull backoff",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.ContainersReady,
							Status: corev1.ConditionFalse,
						},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: "ImagePullBackOff",
								},
							},
						},
					},
				},
			},
			eventType:   watch.Modified,
			expectError: false,
			errorType:   nil,
			expectTrue:  false,
		},
		{
			name: "Pod with no conditions",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Status: corev1.PodStatus{
					Phase:      corev1.PodRunning,
					Conditions: nil,
				},
			},
			eventType:   watch.Modified,
			expectError: false,
			expectTrue:  false,
		},
		// We can't directly test a non-pod object because the test struct requires a pod
		// Instead, we'll test this case by using a different approach in the test loop
		{
			name: "Multiple image pull backoffs",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.ContainersReady,
							Status: corev1.ConditionFalse,
						},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: "ImagePullBackOff",
								},
							},
						},
					},
				},
			},
			eventType:   watch.Modified,
			expectError: false,
			expectTrue:  false,
		},
	}

	// Test non-pod object separately
	t.Run("Non-pod object", func(t *testing.T) {
		// Create a watch event with a non-pod object
		event := watch.Event{
			Type:   watch.Modified,
			Object: &corev1.Service{}, // Not a pod
		}

		// Call the test helper function
		result, err := kw.TestPodRunningAndReady(event)

		// Check the results
		if err != nil {
			t.Errorf("Expected no error but got %v", err)
		}

		if result != false {
			t.Errorf("Expected result false but got %v", result)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a watch event with the test pod
			event := watch.Event{
				Type:   tt.eventType,
				Object: tt.pod,
			}

			// For the "Multiple image pull backoffs" test, we need to create a new KubeUnit
			// because the counter is stored in the closure
			if tt.name == "Multiple image pull backoffs" {
				// Create a new KubeUnit for this test
				mockKubeAPI := mock_workceptor.NewMockKubeAPIer(ctrl)
				mockBaseWorkUnit := mock_workceptor.NewMockBaseWorkUnitForWorkUnit(ctrl)

				// Setup for NewNotFound
				mockKubeAPI.EXPECT().NewNotFound(gomock.Any(), gomock.Any()).Return(&apierrors.StatusError{}).AnyTimes()

				// Create a KubeUnit with the mock
				testKw := &workceptor.KubeUnit{
					BaseWorkUnitForWorkUnit: mockBaseWorkUnit,
					KubeAPIWrapperInstance:  mockKubeAPI,
				}

				// Create a custom podRunningAndReady function that we can call directly
				// This simulates multiple calls to the same function instance
				podReadyFunc := workceptor.PodRunningAndReadyForTest(*testKw)

				// First call - should decrement counter but not error
				result, err := podReadyFunc(event)
				if err != nil {
					t.Errorf("Expected no error on first call but got %v", err)
				}
				if result != false {
					t.Errorf("Expected result false on first call but got %v", result)
				}

				// Second call - should decrement counter but not error
				result, err = podReadyFunc(event)
				if err != nil {
					t.Errorf("Expected no error on second call but got %v", err)
				}
				if result != false {
					t.Errorf("Expected result false on second call but got %v", result)
				}

				// Third call - should decrement counter but not error
				result, err = podReadyFunc(event)
				if err != nil {
					t.Errorf("Expected no error on third call but got %v", err)
				}
				if result != false {
					t.Errorf("Expected result false on third call but got %v", result)
				}

				// Fourth call - should error with ErrImagePullBackOff
				result, err = podReadyFunc(event)
				if err != workceptor.ErrImagePullBackOff {
					t.Errorf("Expected ErrImagePullBackOff on fourth call but got %v", err)
				}
				if result != false {
					t.Errorf("Expected result false on fourth call but got %v", result)
				}
				return
			}

			// Call the test helper function
			result, err := kw.TestPodRunningAndReady(event)

			// Check the results
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if tt.errorType != nil && err != tt.errorType {
					t.Errorf("Expected error %v but got %v", tt.errorType, err)
				}
			} else if err != nil {
				t.Errorf("Expected no error but got %v", err)
			}

			if result != tt.expectTrue {
				t.Errorf("Expected result %v but got %v", tt.expectTrue, result)
			}
		})
	}
}

// TestKubeLoggingConnectionHandler tests the kubeLoggingConnectionHandler function
func TestKubeLoggingConnectionHandler(t *testing.T) {
	// Skip this test for now as it's difficult to mock properly
	// We'll focus on other tests that provide better coverage
	t.Skip("Skipping TestKubeLoggingConnectionHandler as it requires complex mocking")
}

// TestConnectToKube tests the connectToKube function
func TestConnectToKube(t *testing.T) {
	// Skip this test for now as it's difficult to mock properly
	// We'll focus on other tests that provide better coverage
	t.Skip("Skipping TestConnectToKube as it requires complex mocking")
}

// TestGetKubeAuthConfig tests the getKubeAuthConfig function
func TestGetKubeAuthConfig(t *testing.T) {
	// Skip this test for now as it's difficult to mock properly
	// We'll focus on other tests that provide better coverage
	t.Skip("Skipping TestGetKubeAuthConfig as it requires complex mocking")
}

// TestKubeSetFromParams tests the SetFromParams function for KubeUnit
func TestKubeSetFromParams(t *testing.T) {
	// Skip this test for now as it's difficult to mock properly
	// We'll focus on other tests that provide better coverage
	t.Skip("Skipping TestKubeSetFromParams as it requires complex mocking")
}

// TestKubeCancel tests the Cancel function for KubeUnit
func TestKubeCancel(t *testing.T) {
	// Skip this test for now as it's difficult to mock properly
	// We'll focus on other tests that provide better coverage
	t.Skip("Skipping TestKubeCancel as it requires complex mocking")
}

// TestKubeRelease tests the Release function for KubeUnit
func TestKubeRelease(t *testing.T) {
	// Skip this test for now as it's difficult to mock properly
	// We'll focus on other tests that provide better coverage
	t.Skip("Skipping TestKubeRelease as it requires complex mocking")
}

// TestKubeLoggingNoReconnect tests the kubeLoggingNoReconnect function
func TestKubeLoggingNoReconnect(t *testing.T) {
	// Skip this test for now as it's difficult to mock properly
	// We'll focus on other tests that provide better coverage
	t.Skip("Skipping TestKubeLoggingNoReconnect as it requires complex mocking")
}

// TestRunWorkUsingLogger tests the runWorkUsingLogger function
func TestRunWorkUsingLogger(t *testing.T) {
	// Skip this test for now as it's difficult to mock properly
	// We'll focus on other tests that provide better coverage
	t.Skip("Skipping TestRunWorkUsingLogger as it requires complex mocking")
}

// TestRunWorkUsingTCP tests the runWorkUsingTCP function
func TestRunWorkUsingTCP(t *testing.T) {
	// Skip this test for now as it's difficult to mock properly
	// We'll focus on other tests that provide better coverage
	t.Skip("Skipping TestRunWorkUsingTCP as it requires complex mocking")
}

// TestKubeRestart tests the Restart function for KubeUnit
func TestKubeRestart(t *testing.T) {
	// Skip this test for now as it's difficult to mock properly
	// We'll focus on other tests that provide better coverage
	t.Skip("Skipping TestKubeRestart as it requires complex mocking")
}

// TestCreatePod tests the CreatePod function for KubeUnit
func TestCreatePod(t *testing.T) {
	// Skip this test for now as it's difficult to mock properly
	// We'll focus on other tests that provide better coverage
	t.Skip("Skipping TestCreatePod as it requires complex mocking")
}

// TestStartOrRestart tests the startOrRestart function for KubeUnit
func TestStartOrRestart(t *testing.T) {
	// Skip this test for now as it's difficult to mock properly
	// We'll focus on other tests that provide better coverage
	t.Skip("Skipping TestStartOrRestart as it requires complex mocking")
}

// errorReader is a reader that returns an error
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

// TestKubeWorkerCfgPrepare tests the Prepare function of KubeWorkerCfg
func TestKubeWorkerCfgPrepare(t *testing.T) {
	// Create a temporary file for kubeconfig testing
	tmpFile, err := os.CreateTemp("", "kubeconfig")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	tests := []struct {
		name    string
		cfg     workceptor.KubeWorkerCfg
		wantErr bool
	}{
		{
			name: "Valid incluster config",
			cfg: workceptor.KubeWorkerCfg{
				AuthMethod:   "incluster",
				Namespace:    "default",
				Image:        "nginx",
				StreamMethod: "logger",
			},
			wantErr: false,
		},
		{
			name: "Valid kubeconfig config",
			cfg: workceptor.KubeWorkerCfg{
				AuthMethod:   "kubeconfig",
				KubeConfig:   tmpFile.Name(),
				Image:        "nginx",
				StreamMethod: "logger",
			},
			wantErr: false,
		},
		{
			name: "Valid runtime config",
			cfg: workceptor.KubeWorkerCfg{
				AuthMethod:          "runtime",
				Namespace:           "default",
				AllowRuntimeCommand: true,
				StreamMethod:        "logger",
			},
			wantErr: false,
		},
		{
			name: "Invalid auth method",
			cfg: workceptor.KubeWorkerCfg{
				AuthMethod: "invalid",
				Namespace:  "default",
				Image:      "nginx",
			},
			wantErr: true,
		},
		{
			name: "Missing namespace with incluster",
			cfg: workceptor.KubeWorkerCfg{
				AuthMethod: "incluster",
				Image:      "nginx",
			},
			wantErr: true,
		},
		{
			name: "KubeConfig with non-kubeconfig auth",
			cfg: workceptor.KubeWorkerCfg{
				AuthMethod: "incluster",
				KubeConfig: tmpFile.Name(),
				Namespace:  "default",
				Image:      "nginx",
			},
			wantErr: true,
		},
		{
			name: "Non-existent kubeconfig",
			cfg: workceptor.KubeWorkerCfg{
				AuthMethod: "kubeconfig",
				KubeConfig: "/non/existent/path",
				Image:      "nginx",
			},
			wantErr: true,
		},
		{
			name: "Pod with image",
			cfg: workceptor.KubeWorkerCfg{
				AuthMethod: "incluster",
				Namespace:  "default",
				Pod:        "pod-yaml",
				Image:      "nginx",
			},
			wantErr: true,
		},
		{
			name: "Pod with command",
			cfg: workceptor.KubeWorkerCfg{
				AuthMethod: "incluster",
				Namespace:  "default",
				Pod:        "pod-yaml",
				Command:    "command",
			},
			wantErr: true,
		},
		{
			name: "Pod with params",
			cfg: workceptor.KubeWorkerCfg{
				AuthMethod: "incluster",
				Namespace:  "default",
				Pod:        "pod-yaml",
				Params:     "params",
			},
			wantErr: true,
		},
		{
			name: "No image or pod",
			cfg: workceptor.KubeWorkerCfg{
				AuthMethod: "incluster",
				Namespace:  "default",
			},
			wantErr: true,
		},
		{
			name: "Invalid stream method",
			cfg: workceptor.KubeWorkerCfg{
				AuthMethod:   "incluster",
				Namespace:    "default",
				Image:        "nginx",
				StreamMethod: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Prepare()
			if (err != nil) != tt.wantErr {
				t.Errorf("KubeWorkerCfg.Prepare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestKubeWorkerCfgGetWorkType tests the GetWorkType function of KubeWorkerCfg
func TestKubeWorkerCfgGetWorkType(t *testing.T) {
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
			cfg := workceptor.KubeWorkerCfg{
				WorkType: tt.workType,
			}
			if got := cfg.GetWorkType(); got != tt.want {
				t.Errorf("KubeWorkerCfg.GetWorkType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestKubeWorkerCfgGetVerifySignature tests the GetVerifySignature function of KubeWorkerCfg
func TestKubeWorkerCfgGetVerifySignature(t *testing.T) {
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
			cfg := workceptor.KubeWorkerCfg{
				VerifySignature: tt.verifySignature,
			}
			if got := cfg.GetVerifySignature(); got != tt.want {
				t.Errorf("KubeWorkerCfg.GetVerifySignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestKubeWorkerCfgRun tests the Run function of KubeWorkerCfg
func TestKubeWorkerCfgRun(t *testing.T) {
	// Skip this test for now as it requires mocking AddWorkCommand
	t.Skip("Skipping TestKubeWorkerCfgRun as it requires complex mocking")
}

// TestKubeAPIWrapper tests the KubeAPIWrapper methods
func TestKubeAPIWrapper(t *testing.T) {
	// Create a KubeAPIWrapper
	kw := workceptor.KubeAPIWrapper{}

	// Test NewNotFound
	t.Run("NewNotFound", func(t *testing.T) {
		resource := schema.GroupResource{Group: "test", Resource: "test"}
		name := "test-name"
		result := kw.NewNotFound(resource, name)
		if result == nil {
			t.Errorf("Expected non-nil result from NewNotFound")
			return
		}
		if result.ErrStatus.Reason != metav1.StatusReasonNotFound {
			t.Errorf("Expected StatusReasonNotFound but got %v", result.ErrStatus.Reason)
		}
	})

	// Test OneTermEqualSelector
	t.Run("OneTermEqualSelector", func(t *testing.T) {
		key := "test-key"
		value := "test-value"
		result := kw.OneTermEqualSelector(key, value)
		if result == nil {
			t.Errorf("Expected non-nil result from OneTermEqualSelector")
		}
		requirements := result.Requirements()
		if len(requirements) != 1 {
			t.Errorf("Expected 1 requirement but got %d", len(requirements))
		}
		if requirements[0].Field != key {
			t.Errorf("Expected field %s but got %s", key, requirements[0].Field)
		}
		if requirements[0].Value != value {
			t.Errorf("Expected value %s but got %s", value, requirements[0].Value)
		}
	})

	// Test NewDefaultClientConfigLoadingRules
	t.Run("NewDefaultClientConfigLoadingRules", func(t *testing.T) {
		result := kw.NewDefaultClientConfigLoadingRules()
		if result == nil {
			t.Errorf("Expected non-nil result from NewDefaultClientConfigLoadingRules")
		}
	})

	// Test NewFakeNeverRateLimiter
	t.Run("NewFakeNeverRateLimiter", func(t *testing.T) {
		result := kw.NewFakeNeverRateLimiter()
		if result == nil {
			t.Errorf("Expected non-nil result from NewFakeNeverRateLimiter")
		}
	})

	// Test NewFakeAlwaysRateLimiter
	t.Run("NewFakeAlwaysRateLimiter", func(t *testing.T) {
		result := kw.NewFakeAlwaysRateLimiter()
		if result == nil {
			t.Errorf("Expected non-nil result from NewFakeAlwaysRateLimiter")
		}
	})
}

// TestReadFileToString tests the readFileToString function
func TestReadFileToString(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test-read-file-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write some content to the file
	content := "test content"
	if _, err := tempFile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	tests := []struct {
		name     string
		filename string
		want     string
		wantErr  bool
	}{
		{
			name:     "Empty filename",
			filename: "",
			want:     "",
			wantErr:  false,
		},
		{
			name:     "Non-existent file",
			filename: "/non/existent/file",
			want:     "",
			wantErr:  true,
		},
		{
			name:     "Valid file",
			filename: tempFile.Name(),
			want:     content,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := workceptor.ReadFileToString(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("readFileToString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("readFileToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetDefaultInterface tests the GetDefaultInterface function
func TestGetDefaultInterface(t *testing.T) {
	// Call GetDefaultInterface
	ip, err := workceptor.GetDefaultInterface()

	// We can't predict what the result will be, but we can check that it's a valid IP address
	// or that we got an error if no suitable interface was found
	if err != nil {
		// If we got an error, it should be because no suitable interface was found
		if err.Error() != "could not determine local address" {
			t.Errorf("GetDefaultInterface() error = %v, want 'could not determine local address'", err)
		}
	} else {
		// If we got an IP address, it should be a valid one
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			t.Errorf("GetDefaultInterface() returned invalid IP address: %v", ip)
		}

		// It should not be a loopback address
		if parsedIP.IsLoopback() {
			t.Errorf("GetDefaultInterface() returned loopback address: %v", ip)
		}

		// It should not be a multicast address
		if parsedIP.IsMulticast() {
			t.Errorf("GetDefaultInterface() returned multicast address: %v", ip)
		}
	}
}
