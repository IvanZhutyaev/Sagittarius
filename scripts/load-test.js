// k6 load test script for Sagittarius
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '30s', target: 100 },  // Ramp up to 100 users
    { duration: '1m', target: 100 },    // Stay at 100 users
    { duration: '30s', target: 500 },  // Ramp up to 500 users
    { duration: '1m', target: 500 },   // Stay at 500 users
    { duration: '30s', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<100'], // 95% of requests should be below 100ms
    errors: ['rate<0.01'],            // Error rate should be less than 1%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const userId = `test-user-${Math.floor(Math.random() * 1000)}`;
  const auctionId = 'test-auction-1';
  const amount = 1000 + Math.floor(Math.random() * 5000);
  const idempotencyKey = `test-key-${Date.now()}-${Math.random()}`;

  const payload = JSON.stringify({
    auction_id: auctionId,
    amount: amount,
    idempotency_key: idempotencyKey,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-User-ID': userId,
      'X-Idempotency-Key': idempotencyKey,
    },
  };

  const res = http.post(`${BASE_URL}/api/v1/bids`, payload, params);

  const success = check(res, {
    'status is 200 or 400': (r) => r.status === 200 || r.status === 400,
    'response time < 100ms': (r) => r.timings.duration < 100,
    'has bid_id or error': (r) => {
      const body = JSON.parse(r.body);
      return body.bid_id !== undefined || body.error !== undefined;
    },
  });

  errorRate.add(!success);

  sleep(0.1); // 100ms between requests
}

