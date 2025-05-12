package utils_test

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ansible/receptor/pkg/utils"
)

type fields struct {
	Ctx         context.Context
	JcCancel    context.CancelFunc
	Wg          *sync.WaitGroup
	JcRunning   bool
	RunningLock *sync.Mutex
}

func setupGoodFields() fields {
	goodCtx, goodCancel := context.WithCancel(context.Background())
	goodFields := &fields{
		Ctx:         goodCtx,
		JcCancel:    goodCancel,
		Wg:          &sync.WaitGroup{},
		JcRunning:   true,
		RunningLock: &sync.Mutex{},
	}

	return *goodFields
}

func TestJobContextRunning(t *testing.T) {
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "Positive",
			fields: setupGoodFields(),
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &utils.JobContext{
				Ctx:         tt.fields.Ctx,
				JcCancel:    tt.fields.JcCancel,
				Wg:          tt.fields.Wg,
				JcRunning:   tt.fields.JcRunning,
				RunningLock: tt.fields.RunningLock,
			}
			if got := mw.Running(); got != tt.want {
				t.Errorf("JobContext.Running() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobContextCancel(t *testing.T) {
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name:   "Positive",
			fields: setupGoodFields(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &utils.JobContext{
				Ctx:         tt.fields.Ctx,
				JcCancel:    tt.fields.JcCancel,
				Wg:          tt.fields.Wg,
				JcRunning:   tt.fields.JcRunning,
				RunningLock: tt.fields.RunningLock,
			}
			mw.Cancel()
		})
	}
}

func TestJobContextValue(t *testing.T) {
	type args struct {
		key interface{}
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   interface{}
	}{
		{
			name:   "Positive",
			fields: setupGoodFields(),
			want:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &utils.JobContext{
				Ctx:         tt.fields.Ctx,
				JcCancel:    tt.fields.JcCancel,
				Wg:          tt.fields.Wg,
				JcRunning:   tt.fields.JcRunning,
				RunningLock: tt.fields.RunningLock,
			}
			if got := mw.Value(tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("JobContext.Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobContextDeadline(t *testing.T) {
	tests := []struct {
		name     string
		fields   fields
		wantTime time.Time
		wantOk   bool
	}{
		{
			name:     "Positive",
			fields:   setupGoodFields(),
			wantTime: time.Time{},
			wantOk:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &utils.JobContext{
				Ctx:         tt.fields.Ctx,
				JcCancel:    tt.fields.JcCancel,
				Wg:          tt.fields.Wg,
				JcRunning:   tt.fields.JcRunning,
				RunningLock: tt.fields.RunningLock,
			}
			gotTime, gotOk := mw.Deadline()
			if !reflect.DeepEqual(gotTime, tt.wantTime) {
				t.Errorf("JobContext.Deadline() gotTime = %v, want %v", gotTime, tt.wantTime)
			}
			if gotOk != tt.wantOk {
				t.Errorf("JobContext.Deadline() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestJobContextErr(t *testing.T) {
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "Positive",
			fields:  setupGoodFields(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &utils.JobContext{
				Ctx:         tt.fields.Ctx,
				JcCancel:    tt.fields.JcCancel,
				Wg:          tt.fields.Wg,
				JcRunning:   tt.fields.JcRunning,
				RunningLock: tt.fields.RunningLock,
			}
			if err := mw.Err(); (err != nil) != tt.wantErr {
				t.Errorf("JobContext.Err() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobContextWait(t *testing.T) {
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name:   "Positive",
			fields: setupGoodFields(),
		},
		{
			name: "Nil WaitGroup",
			fields: fields{
				Ctx:         context.Background(),
				JcCancel:    func() {},
				Wg:          nil,
				JcRunning:   false,
				RunningLock: &sync.Mutex{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &utils.JobContext{
				Ctx:         tt.fields.Ctx,
				JcCancel:    tt.fields.JcCancel,
				Wg:          tt.fields.Wg,
				JcRunning:   tt.fields.JcRunning,
				RunningLock: tt.fields.RunningLock,
			}
			mw.Wait()
		})
	}
}

func TestJobContextDone(t *testing.T) {
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name:   "Positive",
			fields: setupGoodFields(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &utils.JobContext{
				Ctx:         tt.fields.Ctx,
				JcCancel:    tt.fields.JcCancel,
				Wg:          tt.fields.Wg,
				JcRunning:   tt.fields.JcRunning,
				RunningLock: tt.fields.RunningLock,
			}
			done := mw.Done()
			if done == nil {
				t.Errorf("JobContext.Done() returned nil")
			}
		})
	}
}

func TestJobContextWorkerDone(t *testing.T) {
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Positive",
			fields: func() fields {
				f := setupGoodFields()
				f.Wg.Add(1)
				return f
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &utils.JobContext{
				Ctx:         tt.fields.Ctx,
				JcCancel:    tt.fields.JcCancel,
				Wg:          tt.fields.Wg,
				JcRunning:   tt.fields.JcRunning,
				RunningLock: tt.fields.RunningLock,
			}
			mw.WorkerDone()
		})
	}
}

func TestJobContextNewJob(t *testing.T) {
	type args struct {
		ctx             context.Context
		workers         int
		returnIfRunning bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "New job with nil RunningLock",
			fields: fields{
				Ctx:         nil,
				JcCancel:    nil,
				Wg:          nil,
				JcRunning:   false,
				RunningLock: nil,
			},
			args: args{
				ctx:             context.Background(),
				workers:         1,
				returnIfRunning: false,
			},
			want: true,
		},
		{
			name: "New job not running",
			fields: fields{
				Ctx:         context.Background(),
				JcCancel:    func() {},
				Wg:          &sync.WaitGroup{},
				JcRunning:   false,
				RunningLock: &sync.Mutex{},
			},
			args: args{
				ctx:             context.Background(),
				workers:         2,
				returnIfRunning: false,
			},
			want: true,
		},
		{
			name: "New job already running, return if running",
			fields: fields{
				Ctx:         context.Background(),
				JcCancel:    func() {},
				Wg:          &sync.WaitGroup{},
				JcRunning:   true,
				RunningLock: &sync.Mutex{},
			},
			args: args{
				ctx:             context.Background(),
				workers:         1,
				returnIfRunning: true,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &utils.JobContext{
				Ctx:         tt.fields.Ctx,
				JcCancel:    tt.fields.JcCancel,
				Wg:          tt.fields.Wg,
				JcRunning:   tt.fields.JcRunning,
				RunningLock: tt.fields.RunningLock,
			}
			if got := mw.NewJob(tt.args.ctx, tt.args.workers, tt.args.returnIfRunning); got != tt.want {
				t.Errorf("JobContext.NewJob() = %v, want %v", got, tt.want)
			}

			// Clean up
			if mw.JcCancel != nil {
				mw.JcCancel()
			}
		})
	}
}

func TestJobContextNewJobCancelExisting(t *testing.T) {
	// Create a JobContext with a running job
	mw := &utils.JobContext{
		RunningLock: &sync.Mutex{},
	}

	// Start a job with 1 worker
	mw.NewJob(context.Background(), 1, false)

	// Verify the job is running
	if !mw.Running() {
		t.Errorf("Expected job to be running")
	}

	// Create a channel to signal when the worker is done
	workerDone := make(chan struct{})

	// Start a worker that will sleep for a short time
	go func() {
		defer close(workerDone)
		time.Sleep(100 * time.Millisecond)
		mw.WorkerDone()
	}()

	// Start a new job before the worker is done
	// This should cancel the existing job
	mw.NewJob(context.Background(), 1, false)

	// Wait for the worker to be done
	<-workerDone

	// Clean up
	mw.Cancel()
}

func TestJobContextIntegration(t *testing.T) {
	// Create a JobContext
	mw := &utils.JobContext{}

	// Start a job with 3 workers
	mw.NewJob(context.Background(), 3, false)

	// Verify the job is running
	if !mw.Running() {
		t.Errorf("Expected job to be running")
	}

	// Start 3 workers
	for i := 0; i < 3; i++ {
		go func(id int) {
			// Simulate work
			time.Sleep(100 * time.Millisecond)

			// Signal that the worker is done
			mw.WorkerDone()
		}(i)
	}

	// Wait for all workers to complete
	mw.Wait()

	// The job might still be running for a short time after all workers complete
	// because the goroutine that sets JcRunning to false might not have run yet.
	// Wait a short time for it to complete.
	time.Sleep(100 * time.Millisecond)

	// Verify the job is no longer running
	if mw.Running() {
		t.Errorf("Expected job to be not running after all workers complete")
	}
}
