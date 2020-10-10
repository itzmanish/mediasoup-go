package mediasoup

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"

	uuid "github.com/satori/go.uuid"
)

const VERSION = "3.6.12"

type WorkerLogLevel string

const (
	WorkerLogLevel_Debug WorkerLogLevel = "debug"
	WorkerLogLevel_Warn                 = "warn"
	WorkerLogLevel_Error                = "error"
	WorkerLogLevel_None                 = "none"
)

type WorkerLogTag string

const (
	WorkerLogTag_INFO      WorkerLogTag = "info"
	WorkerLogTag_ICE                    = "ice"
	WorkerLogTag_DTLS                   = "dtls"
	WorkerLogTag_RTP                    = "rtp"
	WorkerLogTag_SRTP                   = "srtp"
	WorkerLogTag_RTCP                   = "rtcp"
	WorkerLogTag_RTX                    = "rtx"
	WorkerLogTag_BWE                    = "bwe"
	WorkerLogTag_Score                  = "score"
	WorkerLogTag_Simulcast              = "simulcast"
	WorkerLogTag_SVC                    = "svc"
	WorkerLogTag_SCTP                   = "sctp"
	WorkerLogTag_Message                = "message"
)

type WorkerSettings struct {
	/**
	 * Logging level for logs generated by the media worker subprocesses (check
	 * the Debugging documentation). Valid values are 'debug', 'warn', 'error' and
	 * 'none'. Default 'error'.
	 */
	LogLevel WorkerLogLevel

	/**
	 * Log tags for debugging. Check the meaning of each available tag in the
	 * Debugging documentation.
	 */
	LogTags []WorkerLogTag

	/**
	 * Minimun RTC port for ICE, DTLS, RTP, etc. Default 10000.
	 */
	RTCMinPort uint16

	/**
	 * Maximum RTC port for ICE, DTLS, RTP, etc. Default 59999.
	 */
	RTCMaxPort uint16

	/**
	 * Path to the DTLS public certificate file in PEM format. If unset, a
	 * certificate is dynamically created.
	 */
	DTLSCertificateFile string

	/**
	 * Path to the DTLS certificate private key file in PEM format. If unset, a
	 * certificate is dynamically created.
	 */
	DTLSPrivateKeyFile string

	/**
	 * Custom application data.
	 */
	AppData H
}

func (w WorkerSettings) Args() []string {
	args := []string{fmt.Sprintf("--logLevel=%s", w.LogLevel)}

	for _, logTag := range w.LogTags {
		args = append(args, fmt.Sprintf("--logTags=%s", logTag))
	}

	args = append(args, fmt.Sprintf("--rtcMinPort=%d", w.RTCMinPort))
	args = append(args, fmt.Sprintf("--rtcMaxPort=%d", w.RTCMaxPort))

	if len(w.DTLSCertificateFile) > 0 && len(w.DTLSPrivateKeyFile) > 0 {
		args = append(args,
			"--dtlsCertificateFile="+w.DTLSCertificateFile,
			"--dtlsPrivateKeyFile="+w.DTLSPrivateKeyFile,
		)
	}

	return args
}

type WorkerUpdateableSettings struct {
	/**
	 * Logging level for logs generated by the media worker subprocesses (check
	 * the Debugging documentation). Valid values are 'debug', 'warn', 'error' and
	 * 'none'. Default 'error'.
	 */
	LogLevel WorkerLogLevel `json:"logLevel,omitempty"`

	/**
	 * Log tags for debugging. Check the meaning of each available tag in the
	 * Debugging documentation.
	 */
	LogTags []WorkerLogTag `json:"logTags,omitempty"`
}

/**
 * An object with the fields of the uv_rusage_t struct.
 *
 * - http//docs.libuv.org/en/v1.x/misc.html#c.uv_rusage_t
 * - https//linux.die.net/man/2/getrusage
 */
type WorkerResourceUsage struct {
	/**
	 * User CPU time used (in ms).
	 */
	RU_Utime float32 `json:"ru_utime,omitempty"`

	/**
	 * System CPU time used (in ms).
	 */
	RU_Stime float32 `json:"ru_stime,omitempty"`

	/**
	 * Maximum resident set size.
	 */
	RU_Maxrss int `json:"ru_maxrss,omitempty"`

	/**
	 * Integral shared memory size.
	 */
	RU_Ixrss int `json:"ru_ixrss,omitempty"`

	/**
	 * Integral unshared data size.
	 */
	RU_Idrss int `json:"ru_idrss,omitempty"`

	/**
	 * Integral unshared stack size.
	 */
	RU_Isrss int `json:"ru_isrss,omitempty"`

	/**
	 * Page reclaims (soft page faults).
	 */
	RU_Minflt int `json:"ru_minflt,omitempty"`

	/**
	 * Page faults (hard page faults).
	 */
	RU_Majflt int `json:"ru_majflt,omitempty"`

	/**
	 * Swaps.
	 */
	RU_Nswap int `json:"ru_nswap,omitempty"`

	/**
	 * Block input operations.
	 */
	RU_Inblock int `json:"ru_inblock,omitempty"`

	/**
	 * Block output operations.
	 */
	RU_Oublock int `json:"ru_oublock,omitempty"`

	/**
	 * IPC messages sent.
	 */
	RU_Msgsnd int `json:"ru_msgsnd,omitempty"`

	/**
	 * IPC messages received.
	 */
	RU_Msgrcv int `json:"ru_msgrcv,omitempty"`

	/**
	 * Signals received.
	 */
	RU_Nsignals int `json:"ru_nsignals,omitempty"`

	/**
	 * Voluntary context switches.
	 */
	RU_Nvcsw int `json:"ru_nvcsw,omitempty"`

	/**
	 * Involuntary context switches.
	 */
	RU_Nivcsw int `json:"ru_nivcsw,omitempty"`
}

var WorkerBin string = os.Getenv("MEDIASOUP_WORKER_BIN")

func init() {
	if len(WorkerBin) == 0 {
		buildType := os.Getenv("MEDIASOUP_BUILDTYPE")

		if buildType != "Debug" {
			buildType = "Release"
		}

		if runtime.GOOS == "windows" {
			homeDir, _ := os.UserHomeDir()
			WorkerBin = filepath.Join(homeDir, "AppData", "Roaming", "npm", "node_modules",
				"mediasoup", "worker", "out", buildType, "mediasoup-worker")
		} else {
			WorkerBin = filepath.Join("/usr/local/lib/node_modules/mediasoup/worker/out", buildType, "mediasoup-worker")
		}
	}
}

type Option func(w *WorkerSettings)

type Worker struct {
	IEventEmitter
	// Worker logger.
	logger Logger
	// mediasoup-worker child process.
	child *exec.Cmd
	// Worker process PID.
	pid int
	// Channel instance.
	channel *Channel
	// PayloadChannel instance.
	payloadChannel *PayloadChannel
	// Closed flag.
	closed bool
	// Custom app data.
	appData H
	// Routers map.
	routers sync.Map
	// Observer instance.
	observer IEventEmitter

	// spawnDone indices child is started
	spawnDone bool
}

func NewWorker(options ...Option) (worker *Worker, err error) {
	logger := NewLogger("Worker")
	settings := &WorkerSettings{
		LogLevel:   WorkerLogLevel_Error,
		RTCMinPort: 10000,
		RTCMaxPort: 59999,
		AppData:    H{},
	}

	for _, option := range options {
		option(settings)
	}

	logger.Debug("constructor()")

	producerPair, err := createSocketPair()
	if err != nil {
		return
	}
	consumerPair, err := createSocketPair()
	if err != nil {
		return
	}
	payloadProducerPair, err := createSocketPair()
	if err != nil {
		return
	}
	payloadConsumerPair, err := createSocketPair()
	if err != nil {
		return
	}

	producerSocket, err := fileToConn(producerPair[0])
	if err != nil {
		return
	}
	consumerSocket, err := fileToConn(consumerPair[0])
	if err != nil {
		return
	}
	payloadProducerSocket, err := fileToConn(payloadProducerPair[0])
	if err != nil {
		return
	}
	payloadConsumerSocket, err := fileToConn(payloadConsumerPair[0])
	if err != nil {
		return
	}

	logger.Debug("spawning worker process: %s %s", WorkerBin, strings.Join(settings.Args(), " "))

	child := exec.Command(WorkerBin, settings.Args()...)
	child.ExtraFiles = []*os.File{producerPair[1], consumerPair[1], payloadProducerPair[1], payloadConsumerPair[1]}
	child.Env = []string{"MEDIASOUP_VERSION=" + VERSION}

	stderr, err := child.StderrPipe()
	if err != nil {
		return
	}
	stdout, err := child.StdoutPipe()
	if err != nil {
		return
	}
	if err = child.Start(); err != nil {
		return
	}

	pid := child.Process.Pid
	channel := newChannel(producerSocket, consumerSocket, pid)
	payloadChannel := newPayloadChannel(payloadProducerSocket, payloadConsumerSocket)
	workerLogger := NewLogger(fmt.Sprintf("worker[pid:%d]", pid))

	go func() {
		r := bufio.NewReader(stderr)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				break
			}
			workerLogger.Error("(stderr) %s", line)
		}
	}()

	go func() {
		r := bufio.NewReader(stdout)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				break
			}
			workerLogger.Debug("(stdout) %s", line)
		}
	}()

	worker = &Worker{
		IEventEmitter:  NewEventEmitter(),
		logger:         logger,
		child:          child,
		pid:            pid,
		channel:        channel,
		payloadChannel: payloadChannel,
		appData:        settings.AppData,
		observer:       NewEventEmitter(),
	}

	channel.Once(strconv.Itoa(pid), func(event string) {
		if !worker.spawnDone && event == "running" {
			worker.spawnDone = true
			logger.Debug("worker process running [pid:%d]", pid)
			worker.Emit("@success")
		}
	})

	go worker.child.Wait()

	return
}

func (w *Worker) wait() {
	err := w.child.Wait()

	w.child = nil
	w.Close()

	code, signal := 0, ""

	if exiterr, ok := err.(*exec.ExitError); ok {
		// The worker has exited with an exit code != 0
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			code = status.ExitStatus()

			if status.Signaled() {
				signal = status.Signal().String()
			} else if status.Stopped() {
				signal = status.StopSignal().String()
			}
		}
	}

	if !w.spawnDone {
		w.spawnDone = true

		if code == 42 {
			w.logger.Error("worker process failed due to wrong settings [pid:%d]", w.pid)
			w.Emit("@failure", NewTypeError("wrong settings"))
		} else {
			w.logger.Error("worker process failed unexpectedly [pid:%d, code:%d, signal:%s]",
				w.pid, code, signal)
			w.Emit("@failure", fmt.Errorf(`[pid:%d, code:%d, signal:%s]`, w.pid, code, signal))
		}
	} else {
		w.logger.Error("worker process died unexpectedly [pid:%d, code:%d, signal:%s]", w.pid, code, signal)
		w.SafeEmit("died", fmt.Errorf("[pid:%d, code:%d, signal:%s]", w.pid, code, signal))
	}
}

/**
 * Worker process identifier (PID).
 */
func (w *Worker) Pid() int {
	return w.pid
}

/**
 * Whether the Worker is closed.
 */
func (w *Worker) Closed() bool {
	return w.closed
}

// Observer
func (w *Worker) Observer() IEventEmitter {
	return w.observer
}

/**
 * Close the Worker.
 */
func (w *Worker) Close() {
	if w.closed {
		return
	}

	w.logger.Debug("close()")

	w.closed = true

	// Kill the worker process.
	if w.child != nil {
		w.child.Process.Signal(syscall.SIGTERM)
		w.child = nil
	}

	// Close the Channel instance.
	w.channel.Close()

	// Close the PayloadChannel instance.
	w.payloadChannel.Close()

	// Close every Router.
	w.routers.Range(func(key, value interface{}) bool {
		router := value.(*Router)
		router.workerClosed()
		return true
	})
	w.routers = sync.Map{}

	// Emit observer event.
	w.observer.SafeEmit("close")
}

// Dump Worker.
func (w *Worker) Dump() (data []byte, err error) {
	w.logger.Debug("dump()")

	resp := w.channel.Request("worker.dump", nil)

	return resp.Data(), resp.Err()
}

/**
 * Get mediasoup-worker process resource usage.
 */
func (w *Worker) GetResourceUsage() (usage WorkerResourceUsage, err error) {
	w.logger.Debug("getResourceUsage()")

	resp := w.channel.Request("worker.getResourceUsage", nil)
	err = resp.Unmarshal(&usage)

	return
}

// UpdateSettings Update settings.
func (w *Worker) UpdateSettings(settings WorkerUpdateableSettings) error {
	w.logger.Debug("updateSettings()")

	return w.channel.Request("worker.updateSettings", nil, settings).Err()
}

// CreateRouter creates a router.
func (w *Worker) CreateRouter(options RouterOptions) (router *Router, err error) {
	w.logger.Debug("createRouter()")

	internal := internalData{RouterId: uuid.NewV4().String()}

	rsp := w.channel.Request("worker.createRouter", internal, nil)
	if err = rsp.Err(); err != nil {
		return
	}

	rtpCapabilities, err := generateRouterRtpCapabilities(options.MediaCodecs)
	if err != nil {
		return
	}
	data := routerData{RtpCapabilities: rtpCapabilities}
	router = newRouter(routerOptions{
		internal:       internal,
		data:           data,
		channel:        w.channel,
		payloadChannel: w.payloadChannel,
		appData:        options.AppData,
	})

	w.routers.Store(internal.RouterId, router)
	router.On("@close", func() {
		w.routers.Delete(internal.RouterId)
	})
	// Emit observer event.
	w.observer.SafeEmit("newrouter", router)

	return
}

func createSocketPair() (file [2]*os.File, err error) {
	fd, err := syscall.Socketpair(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0)
	if err != nil {
		return
	}
	file[0] = os.NewFile(uintptr(fd[0]), "")
	file[1] = os.NewFile(uintptr(fd[1]), "")

	return
}

func fileToConn(file *os.File) (net.Conn, error) {
	defer file.Close()

	return net.FileConn(file)
}
