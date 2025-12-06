#!/usr/bin/env python3
"""
Analytics Aggregator Service
Consumes events from Kafka and aggregates metrics in ClickHouse
"""

import json
import os
import time
from datetime import datetime, timedelta
from typing import Dict, List

from kafka import KafkaConsumer
from clickhouse_driver import Client
from prometheus_client import Counter, Histogram, Gauge, start_http_server

# Prometheus metrics
events_processed = Counter('analytics_events_processed_total', 'Total events processed', ['event_type'])
processing_duration = Histogram('analytics_processing_duration_seconds', 'Event processing duration')
active_auctions = Gauge('analytics_active_auctions', 'Number of active auctions')
win_rate = Gauge('analytics_win_rate', 'Overall win rate')

class AnalyticsAggregator:
    def __init__(self):
        self.kafka_brokers = os.getenv('KAFKA_BROKERS', 'localhost:9092').split(',')
        self.clickhouse_host = os.getenv('CLICKHOUSE_HOST', 'localhost')
        self.clickhouse_port = int(os.getenv('CLICKHOUSE_PORT', '9000'))
        
        # Initialize ClickHouse client
        self.clickhouse = Client(
            host=self.clickhouse_host,
            port=self.clickhouse_port,
            user=os.getenv('CLICKHOUSE_USER', 'default'),
            password=os.getenv('CLICKHOUSE_PASSWORD', ''),
        )
        
        # Initialize Kafka consumer
        self.consumer = KafkaConsumer(
            'auction-events',
            'bid-submitted',
            bootstrap_servers=self.kafka_brokers,
            group_id='analytics-aggregator',
            value_deserializer=lambda m: json.loads(m.decode('utf-8')),
            auto_offset_reset='latest',
        )
        
        # In-memory aggregations (in production, use Redis or similar)
        self.auction_stats: Dict[str, Dict] = {}
        self.win_counts: Dict[str, int] = {}
        self.bid_counts: Dict[str, int] = {}
        
    def setup_clickhouse(self):
        """Create necessary tables in ClickHouse"""
        self.clickhouse.execute('''
            CREATE TABLE IF NOT EXISTS auction_events (
                event_id String,
                event_type String,
                auction_id String,
                user_id String,
                amount UInt64,
                timestamp DateTime,
                metadata String
            ) ENGINE = MergeTree()
            ORDER BY (auction_id, timestamp)
        ''')
        
        self.clickhouse.execute('''
            CREATE TABLE IF NOT EXISTS auction_metrics (
                auction_id String,
                total_bids UInt32,
                win_rate Float32,
                avg_bid_amount UInt64,
                window_start DateTime,
                window_end DateTime
            ) ENGINE = MergeTree()
            ORDER BY (window_start, auction_id)
        ''')
    
    def process_event(self, event: dict):
        """Process a single event"""
        start_time = time.time()
        event_type = event.get('type') or event.get('@type')
        
        try:
            if 'AuctionStarted' in str(event_type) or 'auction_started' in str(event_type).lower():
                self.handle_auction_started(event)
            elif 'AuctionFinished' in str(event_type) or 'auction_finished' in str(event_type).lower():
                self.handle_auction_finished(event)
            elif 'BidSubmitted' in str(event_type) or 'bid_submitted' in str(event_type).lower():
                self.handle_bid_submitted(event)
            elif 'BidAccepted' in str(event_type) or 'bid_accepted' in str(event_type).lower():
                self.handle_bid_accepted(event)
            
            # Store in ClickHouse
            self.store_event(event)
            
            events_processed.labels(event_type=str(event_type)).inc()
            
        except Exception as e:
            print(f"Error processing event: {e}")
        finally:
            processing_duration.observe(time.time() - start_time)
    
    def handle_auction_started(self, event: dict):
        """Handle auction started event"""
        auction_id = event.get('auction_id') or event.get('auctionId')
        if auction_id:
            self.auction_stats[auction_id] = {
                'started_at': datetime.now(),
                'bids': 0,
                'accepted_bids': 0,
            }
            active_auctions.inc()
    
    def handle_auction_finished(self, event: dict):
        """Handle auction finished event"""
        auction_id = event.get('auction_id') or event.get('auctionId')
        if auction_id and auction_id in self.auction_stats:
            stats = self.auction_stats[auction_id]
            stats['finished_at'] = datetime.now()
            active_auctions.dec()
            
            # Calculate win rate for this auction
            if stats['bids'] > 0:
                auction_win_rate = stats['accepted_bids'] / stats['bids']
                win_rate.set(auction_win_rate)
    
    def handle_bid_submitted(self, event: dict):
        """Handle bid submitted event"""
        auction_id = event.get('auction_id') or event.get('auctionId')
        if auction_id and auction_id in self.auction_stats:
            self.auction_stats[auction_id]['bids'] += 1
    
    def handle_bid_accepted(self, event: dict):
        """Handle bid accepted event"""
        auction_id = event.get('auction_id') or event.get('auctionId')
        if auction_id and auction_id in self.auction_stats:
            self.auction_stats[auction_id]['accepted_bids'] += 1
    
    def store_event(self, event: dict):
        """Store event in ClickHouse"""
        try:
            self.clickhouse.execute(
                'INSERT INTO auction_events VALUES',
                [(
                    event.get('id') or event.get('bid_id') or event.get('auction_id', ''),
                    str(event.get('type') or event.get('@type', 'unknown')),
                    event.get('auction_id') or event.get('auctionId', ''),
                    event.get('user_id') or event.get('userId', ''),
                    int(event.get('amount', 0) or event.get('final_price', 0)),
                    datetime.now(),
                    json.dumps(event),
                )]
            )
        except Exception as e:
            print(f"Error storing event in ClickHouse: {e}")
    
    def aggregate_metrics(self):
        """Periodically aggregate metrics"""
        while True:
            time.sleep(60)  # Aggregate every minute
            
            window_end = datetime.now()
            window_start = window_end - timedelta(minutes=1)
            
            # Aggregate metrics for time window
            # In production, query ClickHouse for aggregation
            print(f"Aggregating metrics for window {window_start} - {window_end}")
    
    def run(self):
        """Main run loop"""
        print("Analytics Aggregator started")
        self.setup_clickhouse()
        
        # Start metrics aggregation in background
        import threading
        aggregation_thread = threading.Thread(target=self.aggregate_metrics, daemon=True)
        aggregation_thread.start()
        
        # Start Prometheus metrics server
        start_http_server(8000)
        
        # Consume events
        for message in self.consumer:
            try:
                event = message.value
                self.process_event(event)
            except Exception as e:
                print(f"Error processing message: {e}")

if __name__ == '__main__':
    aggregator = AnalyticsAggregator()
    aggregator.run()

