package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Request metrics
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
		},
		[]string{"method", "endpoint", "status"},
	)

	RequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// Bid metrics
	BidSubmitted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bids_submitted_total",
			Help: "Total number of bids submitted",
		},
		[]string{"auction_id", "status"},
	)

	BidProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bid_processing_duration_seconds",
			Help:    "Bid processing duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
		},
		[]string{"auction_id"},
	)

	// Auction metrics
	ActiveAuctions = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_auctions_total",
			Help: "Number of active auctions",
		},
	)

	AuctionWinRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "auction_win_rate",
			Help: "Win rate for auctions",
		},
		[]string{"auction_id"},
	)

	// Budget metrics
	BudgetOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "budget_operations_total",
			Help: "Total budget operations",
		},
		[]string{"operation", "status"},
	)

	// Kafka metrics
	KafkaMessagesProduced = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_produced_total",
			Help: "Total Kafka messages produced",
		},
		[]string{"topic"},
	)

	KafkaMessagesConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_consumed_total",
			Help: "Total Kafka messages consumed",
		},
		[]string{"topic", "group_id"},
	)

	KafkaConsumerLag = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_consumer_lag",
			Help: "Kafka consumer lag",
		},
		[]string{"topic", "group_id", "partition"},
	)
)

