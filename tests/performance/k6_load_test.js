import redis from 'k6/x/redis';
import { check, sleep } from 'k6';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.1/index.js';

export const options = {
  scenarios: {
    smoke_test: {
      executor: 'constant-vus',
      vus: 5,
      duration: '5s',
      gracefulStop: '0s',
    },
    load_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '5s', target: 50 },
        { duration: '10s', target: 100 },
        { duration: '5s', target: 0 },
      ],
      startTime: '5s',
    },
    stress_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '5s', target: 100 },
        { duration: '10s', target: 200 },
        { duration: '5s', target: 0 },
      ],
      startTime: '25s',
    },
  },
};

const client = new redis.Client('redis://localhost:8082');

export default function () {
  const isConnected = client.isConnected();
  if (!isConnected) {
    console.error('Redis not connected');
    return;
  }

  // KEY Operations
  const key = `k6_key_${__VU}_${__ITER}`;
  const value = `value_${__VU}_${__ITER}`;

  // SET
  client.set(key, value);

  // GET
  const getRes = client.get(key);
  check(getRes, {
    'GET value is correct': (r) => r === value,
  });

  // Random ZADD
  client.zadd('k6_zset', Math.random(), key);

  // Random READ heavy
  if (Math.random() < 0.8) {
    client.get(key);
  }
}

export function teardown() {
  client.close();
}
