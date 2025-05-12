//go:build !no_workceptor
// +build !no_workceptor

package workceptor

import (
	"bufio"
	"context"
	"net"

	"github.com/ansible/receptor/pkg/utils"
)

// TestConnectToRemote exposes the connectToRemote function for testing
func (rw *remoteUnit) TestConnectToRemote(ctx context.Context) (net.Conn, *bufio.Reader, error) {
	return rw.connectToRemote(ctx)
}

// TestGetConnection exposes the getConnection function for testing
func (rw *remoteUnit) TestGetConnection(ctx context.Context) (net.Conn, *bufio.Reader) {
	return rw.getConnection(ctx)
}

// TestConnectAndRun exposes the connectAndRun function for testing
func (rw *remoteUnit) TestConnectAndRun(ctx context.Context, action actionFunc) error {
	return rw.connectAndRun(ctx, action)
}

// TestGetConnectionAndRun exposes the getConnectionAndRun function for testing
func (rw *remoteUnit) TestGetConnectionAndRun(ctx context.Context, firstTimeSync bool, action actionFunc, failure func()) error {
	return rw.getConnectionAndRun(ctx, firstTimeSync, action, failure)
}

// TestStartRemoteUnit exposes the startRemoteUnit function for testing
func (rw *remoteUnit) TestStartRemoteUnit(ctx context.Context, conn net.Conn, reader *bufio.Reader) error {
	return rw.startRemoteUnit(ctx, conn, reader)
}

// TestCancelOrReleaseRemoteUnit exposes the cancelOrReleaseRemoteUnit function for testing
func (rw *remoteUnit) TestCancelOrReleaseRemoteUnit(ctx context.Context, conn net.Conn, reader *bufio.Reader, release bool) error {
	return rw.cancelOrReleaseRemoteUnit(ctx, conn, reader, release)
}

// TestMonitorRemoteStatus exposes the monitorRemoteStatus function for testing
func (rw *remoteUnit) TestMonitorRemoteStatus(mw *utils.JobContext, forRelease bool) {
	rw.monitorRemoteStatus(mw, forRelease)
}

// TestMonitorRemoteStdout exposes the monitorRemoteStdout function for testing
func (rw *remoteUnit) TestMonitorRemoteStdout(mw *utils.JobContext) {
	rw.monitorRemoteStdout(mw)
}
