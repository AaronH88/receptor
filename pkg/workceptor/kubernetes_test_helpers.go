package workceptor

import (
	"io"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// TestPodRunningAndReady exposes the podRunningAndReady function for testing
func (kw *KubeUnit) TestPodRunningAndReady(event watch.Event) (bool, error) {
	podReadyFunc := podRunningAndReady(*kw)
	return podReadyFunc(event)
}

// TestKubeLoggingConnectionHandler exposes the kubeLoggingConnectionHandler function for testing
func (kw *KubeUnit) TestKubeLoggingConnectionHandler(timestamps bool, sinceTime time.Time) (io.ReadCloser, error) {
	return kw.kubeLoggingConnectionHandler(timestamps, sinceTime)
}

// TestKubeLoggingNoReconnect exposes the kubeLoggingNoReconnect function for testing
func (kw *KubeUnit) TestKubeLoggingNoReconnect(streamWait *sync.WaitGroup, stdout *STDoutWriter, stdoutErr *error) {
	kw.kubeLoggingNoReconnect(streamWait, stdout, stdoutErr)
}

// TestRunWorkUsingLogger exposes the runWorkUsingLogger function for testing
func (kw *KubeUnit) TestRunWorkUsingLogger() {
	kw.runWorkUsingLogger()
}

// TestRunWorkUsingTCP exposes the runWorkUsingTCP function for testing
func (kw *KubeUnit) TestRunWorkUsingTCP() {
	kw.runWorkUsingTCP()
}

// SetStreamMethod sets the streamMethod for testing
func (kw *KubeUnit) SetStreamMethod(method string) {
	kw.streamMethod = method
}

// SetDeletePodOnRestart sets the deletePodOnRestart flag for testing
func (kw *KubeUnit) SetDeletePodOnRestart(delete bool) {
	kw.deletePodOnRestart = delete
}

// SetAuthMethod sets the authMethod for testing
func (kw *KubeUnit) SetAuthMethod(method string) {
	kw.authMethod = method
}

// We can't directly test these functions because they require a StdoutWriter
// Instead, we'll focus on testing the exported functions that use them

// TestCreatePod exposes the CreatePod function for testing
// Note: This is already exported, so we're just providing a consistent naming pattern
func (kw *KubeUnit) TestCreatePod(params map[string]string) error {
	return kw.CreatePod(params)
}

// TestConnectToKube exposes the connectToKube function for testing
func (kw *KubeUnit) TestConnectToKube() error {
	return kw.connectToKube()
}

// TestSetClientset allows setting the clientset for testing
func (kw *KubeUnit) TestSetClientset(clientset *kubernetes.Clientset) {
	kw.clientset = clientset
}

// TestGetKubeAuthConfig exposes the getKubeAuthConfig function for testing
func (kw *KubeUnit) TestGetKubeAuthConfig() error {
	// This is a simplified version that just calls connectToKube
	// since getKubeAuthConfig is not directly accessible
	return kw.connectToKube()
}

// PodRunningAndReadyForTest exposes the podRunningAndReady function for testing
// This is different from TestPodRunningAndReady because it returns the function itself
// rather than calling it, allowing for multiple calls to the same function instance
func PodRunningAndReadyForTest(kw KubeUnit) func(event watch.Event) (bool, error) {
	return podRunningAndReady(kw)
}

// TestConnectUsingIncluster exposes the connectUsingIncluster function for testing
func (kw *KubeUnit) TestConnectUsingIncluster() error {
	return kw.connectUsingIncluster()
}

// TestConnectUsingKubeconfig exposes the connectUsingKubeconfig function for testing
func (kw *KubeUnit) TestConnectUsingKubeconfig() error {
	return kw.connectUsingKubeconfig()
}

// This method is now defined above

// SetAllowRuntimeAuth sets the allowRuntimeAuth flag for testing
func (kw *KubeUnit) SetAllowRuntimeAuth(allow bool) {
	kw.allowRuntimeAuth = allow
}

// SetAllowRuntimeCommand sets the allowRuntimeCommand flag for testing
func (kw *KubeUnit) SetAllowRuntimeCommand(allow bool) {
	kw.allowRuntimeCommand = allow
}

// SetAllowRuntimeParams sets the allowRuntimeParams flag for testing
func (kw *KubeUnit) SetAllowRuntimeParams(allow bool) {
	kw.allowRuntimeParams = allow
}

// SetAllowRuntimePod sets the allowRuntimePod flag for testing
func (kw *KubeUnit) SetAllowRuntimePod(allow bool) {
	kw.allowRuntimePod = allow
}

// SetBaseParams sets the baseParams for testing
func (kw *KubeUnit) SetBaseParams(params string) {
	kw.baseParams = params
}

// SetPod sets the Pod for testing
func (kw *KubeUnit) SetPod(pod *corev1.Pod) {
	kw.Pod = pod
}

// SetClientset sets the clientset for testing
func (kw *KubeUnit) SetClientset(clientset *kubernetes.Clientset) {
	kw.clientset = clientset
}

// TestStartOrRestart exposes the startOrRestart function for testing
func (kw *KubeUnit) TestStartOrRestart() error {
	return kw.startOrRestart()
}
