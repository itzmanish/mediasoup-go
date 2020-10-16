package mediasoup

import (
	"encoding/json"
	"reflect"
	"sync"
	"sync/atomic"
)

type ConsumerOptions struct {
	/**
	 * The id of the Producer to consume.
	 */
	ProducerId string `json:"producerId,omitempty"`

	/**
	 * RTP capabilities of the consuming endpoint.
	 */
	RtpCapabilities RtpCapabilities `json:"rtpCapabilities,omitempty"`

	/**
	 * Whether the Consumer must start in paused mode. Default false.
	 *
	 * When creating a video Consumer, it's recommended to set paused to true,
	 * then transmit the Consumer parameters to the consuming endpoint and, once
	 * the consuming endpoint has created its local side Consumer, unpause the
	 * server side Consumer using the resume() method. This is an optimization
	 * to make it possible for the consuming endpoint to render the video as far
	 * as possible. If the server side Consumer was created with paused false,
	 * mediasoup will immediately request a key frame to the remote Producer and
	 * suych a key frame may reach the consuming endpoint even before it's ready
	 * to consume it, generating “black” video until the device requests a keyframe
	 * by itself.
	 */
	Paused bool `json:"paused,omitempty"`

	/**
	 * Preferred spatial and temporal layer for simulcast or SVC media sources.
	 * If unset, the highest ones are selected.
	 */
	PreferredLayers ConsumerLayers `json:"preferredLayers,omitempty"`

	/**
	 * Custom application data.
	 */
	AppData interface{} `json:"appData,omitempty"`
}

/**
 * Valid types for 'trace' event.
 */
type ConsumerTraceEventType string

const (
	ConsumerTraceEventType_Rtp      ConsumerTraceEventType = "rtp"
	ConsumerTraceEventType_Keyframe                        = "keyframe"
	ConsumerTraceEventType_Nack                            = "nack"
	ConsumerTraceEventType_Pli                             = "pli"
	ConsumerTraceEventType_Fir                             = "fir"
)

/**
 * 'trace' event data.
 */
type ConsumerTraceEventData struct {
	/**
	 * Trace type.
	 */
	Type ConsumerTraceEventType `json:"type,omitempty"`

	/**
	 * Event timestamp.
	 */
	Timestamp int64 `json:"timestamp,omitempty"`

	/**
	 * Event direction, "in" | "out".
	 */
	Direction string `json:"direction,omitempty"`

	/**
	 * Per type information.
	 */
	Info H `json:"info,omitempty"`
}

type ConsumerScore struct {
	/**
	 * The score of the RTP stream of the consumer.
	 */
	Score uint32 `json:"score,omitempty"`

	/**
	 * The score of the currently selected RTP stream of the producer.
	 */
	ProducerScore uint32 `json:"producerScore,omitempty"`

	/**
	 * The scores of all RTP streams in the producer ordered by encoding (just
	 * useful when the producer uses simulcast).
	 */
	ProducerScores []uint32 `json:"producerScores,omitempty"`
}

type ConsumerLayers struct {
	/**
	 * The spatial layer index (from 0 to N).
	 */
	SpatialLayer uint8 `json:"spatialLayer,omitempty"`

	/**
	 * The temporal layer index (from 0 to N).
	 */
	TemporalLayer uint8 `json:"temporalLayer,omitempty"`
}

type ConsumerStat = ProducerStat

/**
 * Consumer type.
 */
type ConsumerType string

const (
	ConsumerType_Simple    ConsumerType = "simple"
	ConsumerType_Simulcast              = "simulcast"
	ConsumerType_Svc                    = "svc"
	ConsumerType_Pipe                   = "pipe"
)

type consumerParams struct {
	// {
	// 	 routerId: string;
	// 	 transportId: string;
	// 	 consumerId: string;
	// 	 producerId: string;
	// };
	internal        internalData
	data            consumerData
	channel         *Channel
	payloadChannel  *PayloadChannel
	appData         interface{}
	paused          bool
	producerPaused  bool
	score           ConsumerScore
	preferredLayers ConsumerLayers
}

type consumerData struct {
	Kind          MediaKind
	Type          ConsumerType
	RtpParameters RtpParameters
}

/**
 * Consumer
 * @emits transportclose
 * @emits producerclose
 * @emits producerpause
 * @emits producerresume
 * @emits score - (score: ConsumerScore)
 * @emits layerschange - (layers: ConsumerLayers | undefined)
 * @emits rtp - (packet: Buffer)
 * @emits trace - (trace: ConsumerTraceEventData)
 * @emits @close
 * @emits @producerclose
 */
type Consumer struct {
	IEventEmitter
	locker          sync.Mutex
	logger          Logger
	internal        internalData
	data            consumerData
	channel         *Channel
	payloadChannel  *PayloadChannel
	appData         interface{}
	paused          bool
	closed          uint32
	producerPaused  bool
	priority        uint32
	score           ConsumerScore
	preferredLayers ConsumerLayers
	currentLayers   ConsumerLayers // Current video layers (just for video with simulcast or SVC).
	observer        IEventEmitter
}

func newConsumer(params consumerParams) *Consumer {
	logger := NewLogger("Consumer")

	logger.Debug("constructor()")

	if params.appData == nil {
		params.appData = H{}
	}

	if reflect.DeepEqual(params.score, ConsumerScore{}) {
		params.score = ConsumerScore{
			Score:          10,
			ProducerScore:  10,
			ProducerScores: []uint32{},
		}
	}

	consumer := &Consumer{
		IEventEmitter:   NewEventEmitter(),
		logger:          logger,
		internal:        params.internal,
		data:            params.data,
		channel:         params.channel,
		payloadChannel:  params.payloadChannel,
		appData:         params.appData,
		paused:          params.paused,
		producerPaused:  params.producerPaused,
		score:           params.score,
		preferredLayers: params.preferredLayers,
		observer:        NewEventEmitter(),
	}

	consumer.handleWorkerNotifications()

	return consumer
}

// Consumer id
func (consumer *Consumer) Id() string {
	return consumer.internal.ConsumerId
}

// Associated Consumer id.
func (consumer *Consumer) ConsumerId() string {
	return consumer.internal.ConsumerId
}

// Associated Producer id.
func (consumer *Consumer) ProducerId() string {
	return consumer.internal.ProducerId
}

// Whether the Consumer is closed.
func (consumer *Consumer) Closed() bool {
	return atomic.LoadUint32(&consumer.closed) > 0
}

// Media kind.
func (consumer *Consumer) Kind() MediaKind {
	return consumer.data.Kind
}

// RTP parameters.
func (consumer *Consumer) RtpParameters() RtpParameters {
	return consumer.data.RtpParameters
}

// Consumer type.
func (consumer *Consumer) Type() ConsumerType {
	return consumer.data.Type
}

// Whether the Consumer is paused.
func (consumer *Consumer) Paused() bool {
	return consumer.paused
}

// Whether the associate Producer is paused.
func (consumer *Consumer) ProducerPaused() bool {
	return consumer.producerPaused
}

// Current priority.
func (consumer *Consumer) Priority() uint32 {
	return consumer.priority
}

// Consumer score with consumer and consumer keys.
func (consumer *Consumer) Score() ConsumerScore {
	return consumer.score
}

// Preferred video layers.
func (consumer *Consumer) PreferredLayers() ConsumerLayers {
	return consumer.preferredLayers
}

// Current video layers.
func (consumer *Consumer) CurrentLayers() ConsumerLayers {
	return consumer.currentLayers
}

// App custom data.
func (consumer *Consumer) AppData() interface{} {
	return consumer.appData
}

/**
 * Observer.
 *
 * @emits close
 * @emits pause
 * @emits resume
 * @emits score - (score: ConsumerScore)
 * @emits layerschange - (layers: ConsumerLayers | undefined)
 * @emits trace - (trace: ConsumerTraceEventData)
 */
func (consumer *Consumer) Observer() IEventEmitter {
	return consumer.observer
}

// Close the Consumer.
func (consumer *Consumer) Close() (err error) {
	if atomic.CompareAndSwapUint32(&consumer.closed, 0, 1) {
		consumer.logger.Debug("close()")

		// Remove notification subscriptions.
		consumer.channel.RemoveAllListeners(consumer.internal.ConsumerId)
		consumer.payloadChannel.RemoveAllListeners(consumer.internal.ConsumerId)

		consumer.channel.Request("consumer.close", consumer.internal)

		consumer.Emit("@close")

		// Emit observer event.
		consumer.observer.SafeEmit("close")
	}
	return
}

// Transport was closed.
func (consumer *Consumer) transportClosed() {
	if atomic.CompareAndSwapUint32(&consumer.closed, 0, 1) {
		consumer.logger.Debug("transportClosed()")

		// Remove notification subscriptions.
		consumer.channel.RemoveAllListeners(consumer.internal.ConsumerId)
		consumer.payloadChannel.RemoveAllListeners(consumer.internal.ConsumerId)

		consumer.SafeEmit("transportclose")

		// Emit observer event.
		consumer.observer.SafeEmit("close")
	}
}

// Dump Consumer.
func (consumer *Consumer) Dump() DumpResult {
	consumer.logger.Debug("dump()")

	resp := consumer.channel.Request("consumer.dump", consumer.internal)

	return NewDumpResult(resp.Data(), resp.Err())
}

// Get Consumer stats.
func (consumer *Consumer) GetStats() (stats []ConsumerStat, err error) {
	consumer.logger.Debug("getStats()")

	resp := consumer.channel.Request("consumer.getStats", consumer.internal)
	err = resp.Unmarshal(&stats)

	return
}

// Pause the Consumer.
func (consumer *Consumer) Pause() (err error) {
	consumer.locker.Lock()
	defer consumer.locker.Unlock()

	consumer.logger.Debug("pause()")

	wasPaused := consumer.paused || consumer.producerPaused

	response := consumer.channel.Request("consumer.pause", consumer.internal)

	if err = response.Err(); err != nil {
		return
	}

	consumer.paused = true

	// Emit observer event.
	if !wasPaused {
		consumer.observer.SafeEmit("pause")
	}

	return
}

// Resume the Consumer.
func (consumer *Consumer) Resume() (err error) {
	consumer.locker.Lock()
	defer consumer.locker.Unlock()

	consumer.logger.Debug("resume()")

	wasPaused := consumer.paused || consumer.producerPaused

	response := consumer.channel.Request("consumer.resume", consumer.internal)

	if err = response.Err(); err != nil {
		return
	}

	consumer.paused = false

	// Emit observer event.
	if wasPaused && !consumer.producerPaused {
		consumer.observer.SafeEmit("resume")
	}

	return
}

// Set preferred video layers.
func (consumer *Consumer) SetPreferredLayers(layers ConsumerLayers) (err error) {
	consumer.logger.Debug("setPreferredLayers()")

	response := consumer.channel.Request("consumer.setPreferredLayers", consumer.internal, layers)
	err = response.Unmarshal(&consumer.preferredLayers)

	return
}

// Set priority.
func (consumer *Consumer) SetPriority(priority uint32) (err error) {
	consumer.logger.Debug("setPriority()")

	response := consumer.channel.Request("consumer.setPriority", consumer.internal, H{"priority": priority})

	var result struct {
		Priority uint32
	}
	if err = response.Unmarshal(&result); err != nil {
		return
	}

	consumer.priority = result.Priority

	return
}

// Unset priority.
func (consumer *Consumer) UnsetPriority() (err error) {
	consumer.logger.Debug("unsetPriority()")

	return consumer.SetPriority(1)
}

// Request a key frame to the Producer.
func (consumer *Consumer) RequestKeyFrame() error {
	consumer.logger.Debug("requestKeyFrame()")

	response := consumer.channel.Request("consumer.requestKeyFrame", consumer.internal)

	return response.Err()
}

/**
 * Enable 'trace' event.
 */
func (consumer *Consumer) EnableTraceEvent(types ...ConsumerTraceEventType) error {
	consumer.logger.Debug("enableTraceEvent()")

	response := consumer.channel.Request("consumer.enableTraceEvent", consumer.internal, H{"types": types})

	return response.Err()
}

func (consumer *Consumer) handleWorkerNotifications() {
	consumer.channel.On(consumer.Id(), func(event string, data []byte) {
		switch event {
		case "producerclose":
			if atomic.CompareAndSwapUint32(&consumer.closed, 0, 1) {
				consumer.channel.RemoveAllListeners(consumer.internal.ConsumerId)

				consumer.Emit("@producerclose")
				consumer.SafeEmit("producerclose")

				// Emit observer event.
				consumer.observer.SafeEmit("close")
			}

		case "producerpause":
			consumer.locker.Lock()
			defer consumer.locker.Unlock()

			if consumer.producerPaused {
				break
			}

			wasPaused := consumer.paused || consumer.producerPaused

			consumer.producerPaused = true

			consumer.SafeEmit("producerpause")

			// Emit observer event.
			if !wasPaused {
				consumer.observer.SafeEmit("pause")
			}

		case "producerresume":
			consumer.locker.Lock()
			defer consumer.locker.Unlock()

			if !consumer.producerPaused {
				break
			}

			wasPaused := consumer.paused || consumer.producerPaused

			consumer.producerPaused = false

			consumer.SafeEmit("producerresume")

			// Emit observer event.
			if wasPaused && !consumer.paused {
				consumer.observer.SafeEmit("resume")
			}

		case "score":
			var score ConsumerScore

			json.Unmarshal(data, &score)

			consumer.score = score

			consumer.SafeEmit("score", score)

			// Emit observer event.
			consumer.observer.SafeEmit("score", score)

		case "layerschange":
			var layers ConsumerLayers

			json.Unmarshal(data, &layers)

			consumer.currentLayers = layers

			consumer.SafeEmit("layerschange", layers)

			// Emit observer event.
			consumer.observer.SafeEmit("layerschange", layers)

		case "trace":
			var trace ConsumerTraceEventData

			json.Unmarshal(data, &trace)

			consumer.SafeEmit("trace", trace)

			// Emit observer event.
			consumer.observer.SafeEmit("trace", trace)

		default:
			consumer.logger.Error(`ignoring unknown event "%s" in channel listener`, event)
		}
	})

	consumer.payloadChannel.On(consumer.Id(), func(event string, data, payload []byte) {
		switch event {
		case "rtp":
			if consumer.Closed() {
				return
			}
			consumer.SafeEmit("rtp", payload)

		default:
			consumer.logger.Error(`ignoring unknown event "%s" in payload channel listener`, event)
		}
	})
}
