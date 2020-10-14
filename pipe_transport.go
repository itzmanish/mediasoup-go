package mediasoup

import (
	"encoding/json"
	"fmt"

	uuid "github.com/satori/go.uuid"
)

type PipeTransportOptions struct {
	/**
	 * Listening IP address.
	 */
	ListenIp TransportListenIp

	/**
	 * Create a SCTP association. Default false.
	 */
	EnableSctp bool

	/**
	 * SCTP streams number.
	 */
	numSctpStreams NumSctpStreams

	/**
	 * Maximum allowed size for SCTP messages sent by DataProducers.
	 * Default 268435456.
	 */
	MaxSctpMessageSize int

	/**
	 * Maximum SCTP send buffer used by DataConsumers.
	 * Default 268435456.
	 */
	SctpSendBufferSize int

	/**
	 * Enable RTX and NACK for RTP retransmission. Useful if both Routers are
	 * located in different hosts and there is packet lost in the link. For this
	 * to work, both PipeTransports must enable this setting. Default false.
	 */
	EnableRtx bool

	/**
	 * Enable SRTP. Useful to protect the RTP and RTCP traffic if both Routers
	 * are located in different hosts. For this to work, connect() must be called
	 * with remote SRTP parameters. Default false.
	 */
	EnableSrtp bool

	/**
	 * Custom application data.
	 */
	AppData interface{}
}

type PipeTransportSpecificStat struct {
	Tuple TransportTuple `json:"tuple"`
}

type pipeTransortData struct {
	tuple          TransportTuple
	sctpParameters SctpParameters
	sctpState      SctpState
	rtx            bool
	srtpParameters SrtpParameters
}

type PipeTransport struct {
	ITransport
	logger          Logger
	internal        internalData
	data            pipeTransortData
	channel         *Channel
	payloadChannel  *PayloadChannel
	getProducerById func(string) *Producer
}

func newPipeTransport(params transportParams, data pipeTransortData) *PipeTransport {
	params.logger = NewLogger("PipeTransport")
	params.data = transportData{
		sctpParameters:  data.sctpParameters,
		sctpState:       data.sctpState,
		isPipeTransport: true,
	}

	transport := &PipeTransport{
		ITransport:      newTransport(params),
		logger:          params.logger,
		internal:        params.internal,
		data:            data,
		channel:         params.channel,
		payloadChannel:  params.payloadChannel,
		getProducerById: params.getProducerById,
	}

	transport.handleWorkerNotifications()

	return transport
}

/**
 * Transport tuple.
 */
func (t PipeTransport) Tuple() TransportTuple {
	return t.data.tuple
}

/**
 * SCTP parameters.
 */
func (t PipeTransport) SctpParameters() SctpParameters {
	return t.data.sctpParameters
}

/**
 * SCTP state.
 */
func (t PipeTransport) SctpState() SctpState {
	return t.data.sctpState
}

/**
 * SRTP parameters.
 */
func (t PipeTransport) SrtpParameters() SrtpParameters {
	return t.data.srtpParameters
}

/**
 * Observer.
 *
 * @override
 * @emits close
 * @emits newproducer - (producer: Producer)
 * @emits newconsumer - (consumer: Consumer)
 * @emits newdataproducer - (dataProducer: DataProducer)
 * @emits newdataconsumer - (dataConsumer: DataConsumer)
 * @emits sctpstatechange - (sctpState: SctpState)
 * @emits trace - (trace: TransportTraceEventData)
 */
func (transport *PipeTransport) Observer() IEventEmitter {
	return transport.ITransport.Observer()
}

/**
 * Close the PipeTransport.
 *
 * @override
 */
func (transport *PipeTransport) Close() {
	if transport.Closed() {
		return
	}

	if len(transport.data.sctpState) > 0 {
		transport.data.sctpState = SctpState_Closed
	}

	transport.ITransport.Close()
}

/**
 * Router was closed.
 *
 * @override
 */
func (transport *PipeTransport) routerClosed() {
	if transport.Closed() {
		return
	}

	if len(transport.data.sctpState) > 0 {
		transport.data.sctpState = SctpState_Closed
	}

	transport.ITransport.routerClosed()
}

/**
 * Provide the PlainTransport remote parameters.
 *
 * @override
 */
func (transport *PipeTransport) Connect(options TransportConnectOptions) (err error) {
	transport.logger.Debug("connect()")

	reqData := TransportConnectOptions{
		Ip:             options.Ip,
		Port:           options.Port,
		SrtpParameters: options.SrtpParameters,
	}
	resp := transport.channel.Request("transport.connect", transport.internal, reqData)

	var data struct {
		Tuple TransportTuple
	}
	if err = resp.Unmarshal(&data); err != nil {
		return
	}

	// Update data.
	transport.data.tuple = data.Tuple

	return nil
}

/**
 * Create a pipe Consumer.
 *
 * @override
 */
func (transport *PipeTransport) Consume(options ConsumerOptions) (consumer *Consumer, err error) {
	transport.logger.Debug("consume()")

	producerId := options.ProducerId
	appData := options.AppData

	producer := transport.getProducerById(producerId)

	if producer == nil {
		err = fmt.Errorf(`Producer with id "%s" not found`, producerId)
		return
	}

	rtpParameters := getPipeConsumerRtpParameters(producer.ConsumableRtpParameters(), transport.data.rtx)
	internal := transport.internal
	internal.ConsumerId = uuid.NewV4().String()
	internal.ProducerId = producerId

	reqData := H{
		"kind":                   producer.Kind(),
		"rtpParameters":          rtpParameters,
		"type":                   "pipe",
		"consumableRtpEncodings": producer.ConsumableRtpParameters().Encodings,
	}
	resp := transport.channel.Request("transport.consume", internal, reqData)

	var status struct {
		Paused         bool
		ProducerPaused bool
	}
	if err = resp.Unmarshal(&status); err != nil {
		return
	}

	consumerData := consumerData{
		Kind:          producer.Kind(),
		RtpParameters: rtpParameters,
		Type:          "pipe",
	}
	consumer = newConsumer(consumerParams{
		internal:       internal,
		data:           consumerData,
		channel:        transport.channel,
		payloadChannel: transport.payloadChannel,
		appData:        appData,
		paused:         status.Paused,
		producerPaused: status.ProducerPaused,
	})

	baseTransport := transport.ITransport.(*Transport)

	baseTransport.consumers.Store(consumer.Id(), consumer)
	consumer.On("@close", func() {
		baseTransport.consumers.Delete(consumer.Id())
	})
	consumer.On("@producerclose", func() {
		baseTransport.consumers.Delete(consumer.Id())
	})

	// Emit observer event.
	transport.Observer().SafeEmit("newconsumer", consumer)

	return
}

func (transport *PipeTransport) handleWorkerNotifications() {
	transport.channel.On(transport.Id(), func(event string, data []byte) {
		switch event {
		case "sctpstatechange":
			var result struct {
				SctpState SctpState
			}
			json.Unmarshal(data, &result)

			transport.data.sctpState = result.SctpState

			transport.SafeEmit("sctpstatechange", result.SctpState)

			// Emit observer event.
			transport.Observer().SafeEmit("sctpstatechange", result.SctpState)

		case "trace":
			var result TransportTraceEventData
			json.Unmarshal(data, &result)

			transport.SafeEmit("trace", result)

			// Emit observer event.
			transport.Observer().SafeEmit("trace", result)

		default:
			transport.logger.Error(`ignoring unknown event "%s"`, event)
		}
	})
}
