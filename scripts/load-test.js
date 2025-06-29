import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const toolExecutions = new Counter('tool_executions');
const toolDuration = new Trend('tool_duration');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 },  // Ramp up to 10 users
    { duration: '1m', target: 50 },   // Ramp up to 50 users
    { duration: '3m', target: 50 },   // Stay at 50 users
    { duration: '1m', target: 100 },  // Ramp up to 100 users
    { duration: '3m', target: 100 },  // Stay at 100 users
    { duration: '1m', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_failed: ['rate<0.1'],     // Error rate < 10%
    http_req_duration: ['p(95)<1000'], // 95% of requests < 1s
    errors: ['rate<0.1'],              // Custom error rate < 10%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || '';

// Helper function to make authenticated requests
function makeRequest(url, method = 'GET', body = null) {
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };
  
  if (AUTH_TOKEN) {
    params.headers['Authorization'] = `Bearer ${AUTH_TOKEN}`;
  }
  
  if (method === 'POST' && body) {
    return http.post(url, JSON.stringify(body), params);
  }
  
  return http.get(url, params);
}

export default function () {
  // Test health endpoint
  let res = makeRequest(`${BASE_URL}/health`);
  check(res, {
    'health check status is 200': (r) => r.status === 200,
    'health check response time < 100ms': (r) => r.timings.duration < 100,
  });
  
  // Test info endpoint
  res = makeRequest(`${BASE_URL}/info`);
  check(res, {
    'info status is 200': (r) => r.status === 200,
    'info has server info': (r) => {
      const body = JSON.parse(r.body);
      return body.server_info !== undefined;
    },
  });
  
  // Test tools listing
  res = makeRequest(`${BASE_URL}/tools`);
  const toolsCheck = check(res, {
    'tools list status is 200': (r) => r.status === 200,
    'tools list is array': (r) => {
      const body = JSON.parse(r.body);
      return Array.isArray(body.tools);
    },
  });
  
  if (!toolsCheck) {
    errorRate.add(1);
  } else {
    errorRate.add(0);
  }
  
  // Test tool execution (list projects)
  const startTime = new Date();
  res = makeRequest(`${BASE_URL}/tools/list_projects`, 'POST', {});
  const duration = new Date() - startTime;
  
  const execCheck = check(res, {
    'tool execution status is 200': (r) => r.status === 200,
    'tool execution has result': (r) => {
      const body = JSON.parse(r.body);
      return body.result !== undefined;
    },
  });
  
  if (execCheck) {
    toolExecutions.add(1);
    toolDuration.add(duration);
  } else {
    errorRate.add(1);
  }
  
  // Test concurrent tool executions
  const batch = [
    ['POST', `${BASE_URL}/tools/list_projects`, {}],
    ['POST', `${BASE_URL}/tools/get_project`, { project_id: '1' }],
    ['POST', `${BASE_URL}/tools/list_hosts`, { project_id: '1' }],
  ];
  
  const responses = http.batch(batch.map(([method, url, body]) => {
    return ['POST', url, JSON.stringify(body), {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': AUTH_TOKEN ? `Bearer ${AUTH_TOKEN}` : '',
      },
    }];
  }));
  
  responses.forEach((response, index) => {
    check(response, {
      [`batch request ${index} status is 200`]: (r) => r.status === 200,
    });
  });
  
  // Random sleep between requests (0.5-2 seconds)
  sleep(Math.random() * 1.5 + 0.5);
}

// Handle test lifecycle
export function setup() {
  // Check if server is accessible
  const res = makeRequest(`${BASE_URL}/health`);
  if (res.status !== 200) {
    throw new Error(`Server not accessible at ${BASE_URL}`);
  }
  
  return { startTime: new Date() };
}

export function teardown(data) {
  console.log(`Test completed. Duration: ${new Date() - data.startTime}ms`);
}